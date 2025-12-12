package transcription

// import (
// 	"bytes"
// 	"encoding/json"
// 	"errors"
// 	"fmt"
// 	"io"
// 	"mime/multipart"
// 	"net/http"
// 	"net/url"
// 	"os"
// 	"strings"
// 	"time"

// 	"github.com/cenkalti/backoff/v4"
// 	"github.com/sirupsen/logrus"
// 	"voice-insights-go/internal/logger"
// )

// var httpClient = &http.Client{
// 	Timeout: 12 * time.Second,
// }

// // -------------------------------
// //  API Response Structs (Exact)
// // -------------------------------

// type PublishSuccessResponse struct {
// 	Code   int    `json:"Code"`
// 	Status string `json:"Status"`
// 	Data   struct {
// 		MediaId          string `json:"MediaId"`
// 		Status           string `json:"Status"`
// 		LanguageId       int    `json:"LanguageId"`
// 		TranscriptionURL string `json:"TranscriptionURL"`
// 		WordsCount       int    `json:"WordsCount"`
// 	} `json:"Data"`
// 	Reason   string `json:"Reason,omitempty"`
// 	UniqueId string `json:"UniqueId,omitempty"`
// }

// type StatusResponse struct {
// 	Code   int    `json:"Code"`
// 	Status string `json:"Status"`
// 	Data   struct {
// 		AudioURL           string `json:"AudioURL"`
// 		LanguageId         int    `json:"LanguageId"`
// 		Status             string `json:"Status"` // Success, Queued, Processing, Failed
// 		TranscriptionTextURL string `json:"TranscriptionTextURL"`
// 		WordsCount         int    `json:"WordsCount"`
// 	} `json:"Data"`
// 	Reason   string `json:"Reason,omitempty"`
// 	UniqueId string `json:"UniqueId,omitempty"`
// }

// // =========================================
// // PUBLIC FUNCTION
// // GetTranscript(audioURL string) string,error
// // =========================================

// func GetTranscript(callURL string) (string, error) {
// 	log := logger.New().WithField("module", "transcription")

// 	transcribeHost := os.Getenv("TRANSCRIBE_URL")
// 	if transcribeHost == "" {
// 		return "", errors.New("TRANSCRIBE_URL not set")
// 	}

// 	log.WithField("call_url", callURL).Info("starting transcription")

// 	// -------------------------------
// 	// 1) Publish /transcribe request
// 	// -------------------------------
// 	mediaID, existingURL, err := publishAudio(callURL, log)
// 	if err != nil {
// 		return "", err
// 	}

// 	// If transcription already exists
// 	if existingURL != "" {
// 		log.WithField("existing_url", existingURL).Info("transcription already exists, downloading text")
// 		return downloadTranscriptText(existingURL)
// 	}

// 	// ------------------------------
// 	// 2) Poll /getstatus until done
// 	// ------------------------------
// 	finalURL, err := pollUntilDone(mediaID, callURL, log)
// 	if err != nil {
// 		return "", err
// 	}

// 	log.WithField("final_url", finalURL).Info("transcription completed, downloading text")

// 	// ------------------------------
// 	// 3) Download transcript text
// 	// ------------------------------
// 	return downloadTranscriptText(finalURL)
// }

// // =========================================
// // POST /transcribe
// // multipart/form-data
// // fields:
// //  - callRecordingLink
// //  - callType
// // =========================================

// func publishAudio(callURL string, log *logrus.Entry) (string, string, error) {
// 	endpoint := fmt.Sprintf("%s/transcribe", os.Getenv("TRANSCRIBE_URL"))

// 	var b bytes.Buffer
// 	w := multipart.NewWriter(&b)

// 	// Required fields
// 	w.WriteField("callRecordingLink", callURL)
// 	w.WriteField("callType", "C2C") // or C2C — but PNS works fine for hackathon

// 	w.Close()

// 	req, err := http.NewRequest("POST", endpoint, &b)
// 	if err != nil {
// 		return "", "", err
// 	}

// 	req.Header.Set("Content-Type", w.FormDataContentType())

// 	var respObj PublishSuccessResponse
// 	err = doJSONRequest(req, &respObj)
// 	if err != nil {
// 		log.WithError(err).Error("transcribe publish failed")
// 		return "", "", err
// 	}

// 	if respObj.Code != 200 {
// 		return "", "", fmt.Errorf("transcribe publish error: code=%d reason=%s",
// 			respObj.Code, respObj.Reason)
// 	}

// 	// If already completed
// 	if respObj.Data.TranscriptionURL != "" &&
// 		strings.TrimSpace(respObj.Data.Status) == "Success" {
// 		return "", respObj.Data.TranscriptionURL, nil
// 	}

// 	// Otherwise queued → return MediaId
// 	return respObj.Data.MediaId, "", nil
// }

// // =========================================
// // GET /getstatus?mediaId=...
// // Poll until Status = Success
// // =========================================

// func pollUntilDone(mediaID string, callURL string, log *logrus.Entry) (string, error) {
// 	baseURL := fmt.Sprintf("%s/getstatus", os.Getenv("TRANSCRIBE_URL"))

// 	// Poll up to ~60 seconds
// 	for i := 0; i < 40; i++ {

// 		time.Sleep(1500 * time.Millisecond)

// 		u, _ := url.Parse(baseURL)
// 		q := u.Query()
// 		q.Set("mediaId", mediaID)
// 		u.RawQuery = q.Encode()

// 		req, _ := http.NewRequest("GET", u.String(), nil)

// 		var s StatusResponse
// 		if err := doJSONRequest(req, &s); err != nil {
// 			log.WithError(err).Warn("polling failed")
// 			continue
// 		}

// 		log.WithFields(logrus.Fields{
// 			"media_id": mediaID,
// 			"status":   s.Data.Status,
// 		}).Info("polling transcription")

// 		switch s.Data.Status {
// 		case "Success":
// 			return s.Data.TranscriptionTextURL, nil

// 		case "Queued", "Processing":
// 			continue

// 		case "Failed":
// 			return "", fmt.Errorf("transcription failed: %s", s.Reason)
// 		}
// 	}

// 	return "", fmt.Errorf("timeout: transcription did not complete")
// }

// // =========================================
// // Common HTTP JSON helper with retry
// // =========================================

// func doJSONRequest(req *http.Request, target interface{}) error {
// 	bo := backoff.NewExponentialBackOff()
// 	bo.MaxElapsedTime = 12 * time.Second

// 	var lastErr error

// 	operation := func() error {
// 		resp, err := httpClient.Do(req)
// 		if err != nil {
// 			lastErr = err
// 			return err
// 		}
// 		defer resp.Body.Close()

// 		body, _ := io.ReadAll(resp.Body)

// 		if resp.StatusCode >= 500 {
// 			lastErr = fmt.Errorf("server error: %s", body)
// 			return lastErr
// 		}

// 		if err := json.Unmarshal(body, target); err != nil {
// 			lastErr = fmt.Errorf("json decode error: %v body=%s", err, string(body))
// 			return lastErr
// 		}

// 		return nil
// 	}

// 	if err := backoff.Retry(operation, bo); err != nil {
// 		return lastErr
// 	}
// 	return nil
// }

// // =========================================
// // Download transcript text from URL
// // =========================================

// func downloadTranscriptText(url string) (string, error) {
// 	resp, err := httpClient.Get(url)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode >= 300 {
// 		b, _ := io.ReadAll(resp.Body)
// 		return "", fmt.Errorf("failed to download transcript: %s", b)
// 	}

// 	data, _ := io.ReadAll(resp.Body)
// 	return string(data), nil
// }
