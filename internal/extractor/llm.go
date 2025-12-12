package extractor

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"net/http"
// 	"os"
// 	"time"

// 	"github.com/cenkalti/backoff/v4"
// 	"voice-insights-go/internal/types"
// )

// // Use USE_MOCK_LLM=true to enable offline demo
// func ExtractWithPrompt(prompt string) (types.Extraction, error) {
// 	if os.Getenv("USE_MOCK_LLM") == "true" {
// 		// deterministic mock
// 		return types.Extraction{
// 			Category:         "pricing",
// 			IsConfused:       true,
// 			Sentiment:        "negative",
// 			EscalationReason: "unexpected fees",
// 			RootCause:        "GST shown at checkout",
// 			NextBestAction:   "Onboarding: call seller in 24h and show fee breakdown",
// 		}, nil
// 	}
// 	apiURL := os.Getenv("LLM_GATEWAY_URL")
// 	apiKey := os.Getenv("LLM_API_KEY")
// 	model := os.Getenv("LLM_MODEL")
// 	if apiURL == "" || apiKey == "" {
// 		return types.Extraction{}, fmt.Errorf("llm gateway not configured")
// 	}
// 	reqBody := map[string]interface{}{
// 		"model": model,
// 		"messages": []map[string]string{
// 			{"role": "user", "content": prompt},
// 		},
// 		"temperature": 0.0,
// 	}
// 	data, _ := json.Marshal(reqBody)
// 	var lastErr error
// 	var out types.Extraction
// 	operation := func() error {
// 		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
// 		defer cancel()
// 		req, _ := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(data))
// 		req.Header.Set("Content-Type", "application/json")
// 		req.Header.Set("Authorization", "Bearer "+apiKey)
// 		client := &http.Client{Timeout: 12 * time.Second}
// 		resp, err := client.Do(req)
// 		if err != nil {
// 			lastErr = err
// 			return err
// 		}
// 		defer resp.Body.Close()
// 		body, _ := io.ReadAll(resp.Body)
// 		if resp.StatusCode >= 500 {
// 			lastErr = fmt.Errorf("llm server error: %s", string(body))
// 			return lastErr
// 		}
// 		// try to decode expected structure first
// 		type choice struct {
// 			Message struct {
// 				Content string `json:"content"`
// 			} `json:"message"`
// 		}
// 		var parsed struct {
// 			Choices []choice `json:"choices"`
// 		}
// 		if err := json.Unmarshal(body, &parsed); err == nil && len(parsed.Choices) > 0 {
// 			content := parsed.Choices[0].Message.Content
// 			// extract JSON substring
// 			start := bytes.Index([]byte(content), []byte("{"))
// 			end := bytes.LastIndex([]byte(content), []byte("}"))
// 			if start >= 0 && end > start {
// 				raw := content[start : end+1]
// 				if err := json.Unmarshal([]byte(raw), &out); err == nil {
// 					lastErr = nil
// 					return nil
// 				}
// 			}
// 		}
// 		// fallback: try to parse raw body for JSON
// 		start := bytes.Index(body, []byte("{"))
// 		end := bytes.LastIndex(body, []byte("}"))
// 		if start >= 0 && end > start {
// 			raw := body[start : end+1]
// 			if err := json.Unmarshal(raw, &out); err == nil {
// 				lastErr = nil
// 				return nil
// 			}
// 		}
// 		lastErr = fmt.Errorf("unexpected llm response: %s", string(body))
// 		return lastErr
// 	}
// 	b := backoff.NewExponentialBackOff()
// 	b.MaxElapsedTime = 20 * time.Second
// 	if err := backoff.Retry(operation, b); err != nil {
// 		return types.Extraction{}, fmt.Errorf("llm extraction failed: %w", lastErr)
// 	}
// 	return out, nil
// }
