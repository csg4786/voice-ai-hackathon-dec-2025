package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
    "strconv"
	"time"

	"github.com/joho/godotenv"
	"voice-insights-go/internal/logger"
	"voice-insights-go/internal/processor"
)

func main() {
	_ = godotenv.Load() // loads .env
    fmt.Println(">> DEBUG: SEARCH_API_URL =", os.Getenv("SEARCH_API_URL"))


	log := logger.New()
	log.WithField("service", "voice-insights-go").Info("starting service")

	mux := http.NewServeMux()

	// --------------------------------------------------------------------
	// HEALTH CHECK
	// --------------------------------------------------------------------
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		logger.New().WithRequest(r).Info("health check")
		fmt.Fprint(w, "ok")
	})

	// --------------------------------------------------------------------
	// /process — main endpoint after removing dataset summaries
	// --------------------------------------------------------------------
	mux.HandleFunc("/process", func(w http.ResponseWriter, r *http.Request) {
		reqLog := logger.New().WithRequest(r).WithField("handler", "process")
		reqLog.Info("process request received")

		audioURL := r.URL.Query().Get("audio_url")
		if audioURL == "" {
            reqLog.Warn("missing audio_url")
			http.Error(w, "missing audio_url", http.StatusBadRequest)
			return
		}

        k := 3
        kstr := r.URL.Query().Get("k")
        if kstr == "" {
            parsedk, kerr := strconv.ParseInt(kstr, 10, 64)
            if kerr == nil {
                k = int(parsedk) // Convert int64 to int if desired
            }
			return
		}

		timeoutSec := 40
		if t := r.URL.Query().Get("timeout_sec"); t != "" {
			fmt.Sscanf(t, "%d", &timeoutSec)
		}

		reqLog = reqLog.WithField("audio_url", audioURL).WithField("timeout_sec", timeoutSec)

		start := time.Now()
		res, err := processor.ProcessSingleCall(audioURL, k, time.Duration(timeoutSec)*2*time.Second)
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

	// --------------------------------------------------------------------
	// SERVER SETUP
	// --------------------------------------------------------------------
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
