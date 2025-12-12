package transcription

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"voice-insights-go/internal/logger"
)

var httpClient = &http.Client{Timeout: 20 * time.Second}

type PublishSuccessResponse struct {
	Code   int    `json:"Code"`
	Status string `json:"Status"`
	Data   struct {
		MediaId          string `json:"MediaId"`
		Status           string `json:"Status"`
		LanguageId       int    `json:"LanguageId"`
		TranscriptionURL string `json:"TranscriptionURL"`
		WordsCount       int    `json:"WordsCount"`
	} `json:"Data"`
	Reason   string `json:"Reason,omitempty"`
	UniqueId string `json:"UniqueId,omitempty"`
}

type StatusResponse struct {
	Code   int    `json:"Code"`
	Status string `json:"Status"`
	Data   struct {
		AudioURL             string `json:"AudioURL"`
		LanguageId           int    `json:"LanguageId"`
		Status               string `json:"Status"`
		TranscriptionTextURL string `json:"TranscriptionTextURL"`
		WordsCount           int    `json:"WordsCount"`
	} `json:"Data"`
	Reason   string `json:"Reason,omitempty"`
	UniqueId string `json:"UniqueId,omitempty"`
}

// GetTranscript: top-level call. Supports mock mode via env USE_MOCK_TRANSCRIBE=true
func GetTranscript(callURL string) (string, error) {
	log := logger.New().WithField("component", "transcription").WithField("call_url", callURL)
	if os.Getenv("USE_MOCK_TRANSCRIBE") == "true" {
		log.Info("USE_MOCK_TRANSCRIBE=true, returning mock transcript")
		return "MOCK TRANSCRIPT: Customer says they face pricing issues and want refund.", nil
	}
	apiHost := os.Getenv("TRANSCRIBE_URL")
	if apiHost == "" {
		log.Error("TRANSCRIBE_URL not set")
		return "", errors.New("TRANSCRIBE_URL not set")
	}
	log.Info("publishing to transcription API", apiHost)
	mediaID, existingURL, err := publish(callURL, apiHost)
	if err != nil {
		log.WithError(err).Error("publish failed")
		return "", err
	}
	if existingURL != "" {
		log.WithField("transcription_url", existingURL).Info("transcription already exists; downloading")
		return download(existingURL)
	}
	finalURL, err := poll(mediaID, apiHost)
	if err != nil {
		log.WithError(err).Error("poll failed")
		return "", err
	}
	log.WithField("final_url", finalURL).Info("download final transcript")
	return download(finalURL)
}

func publish(callURL, host string) (string, string, error) {
	log := logger.New().WithField("component", "transcription.publish").WithField("call_url", callURL)
	endpoint := strings.TrimRight(host, "/") + "/transcribe"
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	_ = w.WriteField("callRecordingLink", callURL)
	_ = w.WriteField("callType", "PNS")
	_ = w.Close()

	req, _ := http.NewRequest("POST", endpoint, &b)
	req.Header.Set("Content-Type", w.FormDataContentType())

	var resp PublishSuccessResponse
	if err := doJSON(req, &resp); err != nil {
		log.WithError(err).Error("publish request failed")
		return "", "", err
	}
	log.WithField("resp_code", resp.Code).WithField("resp_status", resp.Status).WithField("data", resp.Data).Info("publish response")
	if resp.Code != 200 {
		return "", "", fmt.Errorf("transcribe publish error: code=%d reason=%s", resp.Code, resp.Reason)
	}
	if resp.Data.TranscriptionURL != "" && strings.ToLower(resp.Data.Status) == "success" {
		return "", resp.Data.TranscriptionURL, nil
	}
	return resp.Data.MediaId, "", nil
}

func poll(mediaID, host string) (string, error) {
	log := logger.New().WithField("component", "transcription.poll").WithField("media_id", mediaID)
	base := strings.TrimRight(host, "/") + "/getstatus"
	for i := 0; i < 60; i++ {
		time.Sleep(1500 * time.Millisecond)
		u, _ := url.Parse(base)
		q := u.Query()
		q.Set("mediaId", mediaID)
		u.RawQuery = q.Encode()
		req, _ := http.NewRequest("GET", u.String(), nil)
		var s StatusResponse
		if err := doJSON(req, &s); err != nil {
			log.WithError(err).Warnf("status request failed attempt=%d", i)
			continue
		}
		log.WithField("status", s.Data.Status).WithField("words", s.Data.WordsCount).Info("status check")
		switch strings.ToLower(s.Data.Status) {
		case "success":
			return s.Data.TranscriptionTextURL, nil
		case "queued", "processing":
			continue
		case "failed":
			return "", fmt.Errorf("transcription failed: %s", s.Reason)
		}
	}
	return "", fmt.Errorf("transcription timeout")
}

func download(url string) (string, error) {
	log := logger.New().WithField("component", "transcription.download").WithField("url", url)
	resp, err := httpClient.Get(url)
	if err != nil {
		log.WithError(err).Error("failed to download transcript")
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		log.WithField("status", resp.StatusCode).WithField("body", string(b)).Error("download returned error")
		return "", fmt.Errorf("download failed: %s", string(b))
	}
	b, _ := io.ReadAll(resp.Body)
	txt := string(b)
	log.WithField("size_bytes", len(b)).Info("download complete; transcript")
	// optionally truncate log preview to avoid overwhelming console, but full content still logged because you requested full logging
	log.WithField("transcript_full", txt).Debug("transcript content")
	return txt, nil
}

func doJSON(req *http.Request, target interface{}) error {
	log := logger.New().WithField("component", "transcription.http")
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 20 * time.Second
	var lastErr error
	op := func() error {
		log.WithField("method", req.Method).WithField("url", req.URL.String()).Info("calling external transcription API")
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			log.WithError(err).Warn("http request error")
			return err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		log.WithField("status", resp.StatusCode).WithField("body_len", len(body)).Debug("raw response body")
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %s", string(body))
			return lastErr
		}
		if len(body) == 0 {
			lastErr = fmt.Errorf("empty body")
			return lastErr
		}
		if err := json.Unmarshal(body, target); err != nil {
			lastErr = fmt.Errorf("json decode error: %v body=%s", err, string(body))
			log.WithError(lastErr).Warn("json decode failed")
			return lastErr
		}
		return nil
	}
	if err := backoff.Retry(op, bo); err != nil {
		return lastErr
	}
	return nil
}
