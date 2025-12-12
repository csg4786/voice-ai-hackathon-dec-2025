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

var httpClient = &http.Client{Timeout: 12 * time.Second}

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
	if os.Getenv("USE_MOCK_TRANSCRIBE") == "true" {
		// quick mock transcript
		return "MOCK TRANSCRIPT: Customer says they face pricing issues and want refund.", nil
	}
	log := logger.New().WithField("module", "transcription")
	apiHost := os.Getenv("TRANSCRIBE_URL")
	if apiHost == "" {
		return "", errors.New("TRANSCRIBE_URL not set")
	}
	mediaID, existingURL, err := publish(callURL, apiHost)
	if err != nil {
		return "", err
	}
	if existingURL != "" {
		return download(existingURL)
	}
	finalURL, err := poll(mediaID, apiHost)
	if err != nil {
		return "", err
	}
	log.WithField("final_url", finalURL).Info("download final transcript")
	return download(finalURL)
}

func publish(callURL, host string) (string, string, error) {
	endpoint := strings.TrimRight(host, "/") + "/transcribe"
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.WriteField("callRecordingLink", callURL)
	w.WriteField("callType", "PNS")
	_ = w.Close()
	req, _ := http.NewRequest("POST", endpoint, &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	var resp PublishSuccessResponse
	if err := doJSON(req, &resp); err != nil {
		return "", "", err
	}
	if resp.Code != 200 {
		return "", "", fmt.Errorf("transcribe publish error: code=%d reason=%s", resp.Code, resp.Reason)
	}
	if resp.Data.TranscriptionURL != "" && strings.ToLower(resp.Data.Status) == "success" {
		return "", resp.Data.TranscriptionURL, nil
	}
	return resp.Data.MediaId, "", nil
}

func poll(mediaID, host string) (string, error) {
	base := strings.TrimRight(host, "/") + "/getstatus"
	for i := 0; i < 40; i++ {
		time.Sleep(1500 * time.Millisecond)
		u, _ := url.Parse(base)
		q := u.Query()
		q.Set("mediaId", mediaID)
		u.RawQuery = q.Encode()
		req, _ := http.NewRequest("GET", u.String(), nil)
		var s StatusResponse
		if err := doJSON(req, &s); err != nil {
			continue
		}
		switch s.Data.Status {
		case "Success":
			return s.Data.TranscriptionTextURL, nil
		case "Queued", "Processing":
			continue
		case "Failed":
			return "", fmt.Errorf("transcription failed: %s", s.Reason)
		}
	}
	return "", fmt.Errorf("transcription timeout")
}

func download(url string) (string, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("download failed: %s", string(b))
	}
	b, _ := io.ReadAll(resp.Body)
	return string(b), nil
}

func doJSON(req *http.Request, target interface{}) error {
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 12 * time.Second
	var lastErr error
	op := func() error {
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			return err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
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
			return lastErr
		}
		return nil
	}
	if err := backoff.Retry(op, bo); err != nil {
		return lastErr
	}
	return nil
}
