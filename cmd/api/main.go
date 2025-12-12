package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"voice-insights-go/internal/dataset"
	"voice-insights-go/internal/logger"
	"voice-insights-go/internal/processor"
)

func main() {
	_ = godotenv.Load() // loads .env

	log := logger.New()
	log.WithField("service", "voice-insights-go").Info("starting service")

	// load dataset summary into memory
	dataPath := os.Getenv("DATASET_PATH")
	if dataPath == "" {
		dataPath = "Data_Voice_Hackathon_Master.xlsx"
	}
	log.WithField("dataset_path", dataPath).Info("loading dataset summary")
	summary, err := dataset.LoadAndSummarize(dataPath)
	if err != nil {
		log.WithError(err).Fatal("failed to load dataset summary")
	}
	log.WithField("total_calls", summary.TotalCalls).Info("dataset summary loaded")

	mux := http.NewServeMux()

	// health
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		logger.New().WithRequest(r).Info("health check")
		fmt.Fprint(w, "ok")
	})

	// process endpoint
	mux.HandleFunc("/process", func(w http.ResponseWriter, r *http.Request) {
		reqLog := logger.New().WithRequest(r).WithField("handler", "process")
		reqLog.Info("process request received")

		audioURL := r.URL.Query().Get("audio_url")
		if audioURL == "" {
			reqLog.Warn("missing audio_url")
			http.Error(w, "missing audio_url", http.StatusBadRequest)
			return
		}
		timeoutSec := 40
		if t := r.URL.Query().Get("timeout_sec"); t != "" {
			fmt.Sscanf(t, "%d", &timeoutSec)
		}
		reqLog = reqLog.WithField("audio_url", audioURL).WithField("timeout_sec", timeoutSec)

		start := time.Now()
		res, err := processor.ProcessSingleCallWithDataset(audioURL, time.Duration(timeoutSec)*time.Second, summary)
		duration := time.Since(start)
		reqLog.WithField("duration_ms", duration.Milliseconds()).Info("processor finished")
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			reqLog.WithError(err).Warn("processor returned error")
			w.WriteHeader(http.StatusInternalServerError)
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(res); err != nil {
			reqLog.WithError(err).Error("failed to write response")
		}
	})

	// demo endpoint (process first N rows from dataset for quick demo)
	mux.HandleFunc("/demo", func(w http.ResponseWriter, r *http.Request) {
		reqLog := logger.New().WithRequest(r).WithField("handler", "demo")
		reqLog.Info("demo invoked")
		records, err := dataset.Load(dataPath)
		if err != nil {
			reqLog.WithError(err).Error("dataset load error")
			http.Error(w, "dataset load error", 500)
			return
		}
		limit := 5
		if len(records) < limit {
			limit = len(records)
		}
		demo := records[:limit]
		var out []interface{}
		for _, rec := range demo {
			reqLog := reqLog.WithField("demo_call", rec.CallID).WithField("audio_url", rec.AudioURL)
			reqLog.Info("processing demo call")
			res, _ := processor.ProcessSingleCallWithDataset(rec.AudioURL, 25*time.Second, summary)
			out = append(out, res)
		}
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(out)
	})

	addr := fmt.Sprintf(":%s", envOr("PORT", "8080"))
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	log.WithField("addr", addr).Info("listening")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.WithError(err).Fatal("server terminated")
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
