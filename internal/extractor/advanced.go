// package extractor

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"net/http"
// 	"os"
// 	"strings"
// 	"time"

// 	"github.com/cenkalti/backoff/v4"
// 	"voice-insights-go/internal/logger"
// 	"voice-insights-go/internal/types"
// 	"voice-insights-go/internal/dataset"
// )

// func BuildAdvancedPrompt(transcript string, ds dataset.DatasetSummary) string {
// 	dsJSON, _ := json.MarshalIndent(ds, "", "  ")
// 	p := fmt.Sprintf(`You are a senior call QA and insights engine. Use the dataset summary as ground truth (do not invent numbers). Produce ONLY valid JSON matching the schema below.

// Schema:
// {
//   "customer_problem": {
//     "primary_issue": "",
//     "issue_description": "",
//     "urgency_level": "", 
//     "severity": 0
//   },
//   "agent_analysis": {
//     "steps_explained_by_agent": [],
//     "correctness_of_guidance": true,
//     "missed_opportunities": [],
//     "agent_sentiment": "",
//     "compliance_flags": []
//   },
//   "kpi": {
//     "customer_talk_ratio": 0.0,
//     "agent_talk_ratio": 0.0,
//     "silence_seconds": 0,
//     "interruption_count": 0,
//     "frustration_score": 0.0,
//     "confusion_level": 0.0
//   },
//   "should_have_done": {
//     "ideal_resolution_path": "",
//     "recommended_followup": "",
//     "department_owner": ""
//   },
//   "dataset_insights": {
//     "similar_calls_count": 0,
//     "city_trend": "",
//     "vintage_trend": "",
//     "probable_root_cause": ""
//   }
// }

// DATASET SUMMARY:
// %s

// TRANSCRIPT:
// \"\"\"
// %s
// \"\"\"

// Notes:
// - Use dataset numbers when producing dataset_insights fields.
// - Evidence lines may be included inside dataset_insights text fields.
// - Ratings: severity 1 (low) to 5 (critical).
// Return only JSON (no commentary).`, string(dsJSON), transcript)

// 	return p
// }

// func ExtractAdvanced(prompt string) (types.KPIExtraction, error) {
// 	log := logger.New().WithField("component", "extractor-advanced")
// 	// mock
// 	if os.Getenv("USE_MOCK_LLM") == "true" {
// 		log.Info("mock LLM mode ON - returning deterministic KPIExtraction")
// 		var mock types.KPIExtraction
// 		mock.CustomerProblem.PrimaryIssue = "pricing"
// 		mock.CustomerProblem.IssueDescription = "Customer confused by GST and final price."
// 		mock.CustomerProblem.UrgencyLevel = "medium"
// 		mock.CustomerProblem.Severity = 3

// 		mock.AgentAnalysis.StepsExplainedByAgent = []string{"Explained listing fees", "Asked customer to retry checkout"}
// 		mock.AgentAnalysis.CorrectnessOfGuidance = false
// 		mock.AgentAnalysis.MissedOpportunities = []string{"Did not offer immediate refund or voucher", "Did not explain GST break-up"}
// 		mock.AgentAnalysis.AgentSentiment = "neutral"
// 		mock.AgentAnalysis.ComplianceFlags = []string{}

// 		mock.KPI.CustomerTalkRatio = 0.62
// 		mock.KPI.AgentTalkRatio = 0.38
// 		mock.KPI.SilenceSeconds = 2
// 		mock.KPI.InterruptionCount = 1
// 		mock.KPI.FrustrationScore = 0.68
// 		mock.KPI.ConfusionLevel = 0.75

// 		mock.ShouldHaveDone.IdealResolutionPath = "Explain GST + offer 1-click refund or voucher; follow-up by onboarding team."
// 		mock.ShouldHaveDone.RecommendedFollowUp = "Call the seller within 24h to assist listing."
// 		mock.ShouldHaveDone.DepartmentOwner = "onboarding"

// 		mock.DatasetInsights.SimilarCallsCount = 1240
// 		mock.DatasetInsights.CityTrend = "Surat: high GST complaints"
// 		mock.DatasetInsights.VintageTrend = "0-2M sellers: highest confusion (67%)"
// 		mock.DatasetInsights.ProbableRootCause = "Sellers unaware of mandatory GST and fee display at checkout."

// 		return mock, nil
// 	}

// 	apiURL := os.Getenv("LLM_GATEWAY_URL")
// 	apiKey := os.Getenv("LLM_API_KEY")
// 	model := os.Getenv("LLM_MODEL")
// 	if apiURL == "" || apiKey == "" {
// 		log.WithFields(map[string]interface{}{
// 			"api_url_set": apiURL != "",
// 			"api_key_set": apiKey != "",
// 		}).Error("llm gateway not configured")
// 		return types.KPIExtraction{}, fmt.Errorf("llm gateway not configured")
// 	}

// 	reqBody := map[string]interface{}{
// 		"model": model,
// 		"messages": []map[string]string{
// 			{"role": "user", "content": prompt},
// 		},
// 		"temperature": 0.0,
// 	}
// 	data, _ := json.MarshalIndent(reqBody, "", "  ")

// 	log.Debug("LLM request payload (full):\n" + string(data))

// 	var lastErr error
// 	var extracted types.KPIExtraction

// 	op := func() error {
// 		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
// 		defer cancel()
// 		req, _ := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(data))
// 		req.Header.Set("Content-Type", "application/json")
// 		req.Header.Set("Authorization", "Bearer "+apiKey)
// 		client := &http.Client{Timeout: 25 * time.Second}
// 		resp, err := client.Do(req)
// 		if err != nil {
// 			lastErr = err
// 			log.WithError(err).Warn("llm request failed")
// 			return err
// 		}
// 		defer resp.Body.Close()
// 		body, _ := io.ReadAll(resp.Body)
// 		log.WithField("http_status", resp.StatusCode).Debug("llm raw response (full):\n" + string(body))
// 		if resp.StatusCode >= 500 {
// 			lastErr = fmt.Errorf("llm server error: %s", string(body))
// 			return lastErr
// 		}

// 		// Attempt to extract JSON substring
// 		content := extractJSON(string(body))
// 		if content != "" {
// 			log.Debug("extracted JSON payload from LLM (full):\n" + content)
// 			if err := json.Unmarshal([]byte(content), &extracted); err == nil {
// 				lastErr = nil
// 				return nil
// 			} else {
// 				log.WithError(err).Warn("json unmarshal from extracted content failed")
// 			}
// 		}

// 		// Try common "choices" structure
// 		var respObj map[string]interface{}
// 		if err := json.Unmarshal(body, &respObj); err == nil {
// 			if choices, ok := respObj["choices"].([]interface{}); ok && len(choices) > 0 {
// 				if c0, ok := choices[0].(map[string]interface{}); ok {
// 					// openai-style
// 					if message, ok := c0["message"].(map[string]interface{}); ok {
// 						if content, ok := message["content"].(string); ok {
// 							s := content
// 							if j := extractJSON(s); j != "" {
// 								if err := json.Unmarshal([]byte(j), &extracted); err == nil {
// 									lastErr = nil
// 									return nil
// 								} else {
// 									log.WithError(err).Warn("json unmarshal from choices[0].message.content failed")
// 								}
// 							}
// 						}
// 					}
// 					// or completion.text
// 					if text, ok := c0["text"].(string); ok {
// 						if j := extractJSON(text); j != "" {
// 							if err := json.Unmarshal([]byte(j), &extracted); err == nil {
// 								lastErr = nil
// 								return nil
// 							}
// 						}
// 					}
// 				}
// 			}
// 		}

// 		lastErr = fmt.Errorf("unexpected LLM response: %s", string(body))
// 		return lastErr
// 	}

// 	b := backoff.NewExponentialBackOff()
// 	b.MaxElapsedTime = 45 * time.Second
// 	if err := backoff.Retry(op, b); err != nil {
// 		log.WithError(lastErr).Error("llm extract failed after retries")
// 		return types.KPIExtraction{}, fmt.Errorf("llm extract failed: %w", lastErr)
// 	}

// 	log.WithField("parsed_kpi", fmt.Sprintf("%+v", extracted)).Info("parsed KPIExtraction")
// 	return extracted, nil
// }

// // extractJSON attempts to robustly find the first JSON object in a larger string
// func extractJSON(s string) string {
// 	start := strings.Index(s, "{")
// 	end := strings.LastIndex(s, "}")
// 	if start >= 0 && end > start {
// 		return s[start : end+1]
// 	}
// 	return ""
// }





























// package extractor

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"net/http"
// 	"os"
// 	"strings"
// 	"time"

// 	"github.com/cenkalti/backoff/v4"
// 	"voice-insights-go/internal/logger"
// 	"voice-insights-go/internal/types"
// 	"voice-insights-go/internal/dataset"
// )

// func BuildAdvancedPrompt(transcript string, ds dataset.DatasetSummary) string {
// 	dsJSON, _ := json.MarshalIndent(ds, "", "  ")
// 	p := fmt.Sprintf(`You are a senior call QA and insights engine. Use the dataset summary as ground truth (do not invent numbers). Produce ONLY valid JSON matching the schema below.

// Schema:
// {
//   "customer_problem": {
//     "primary_issue": "",
//     "issue_description": "",
//     "urgency_level": "", 
//     "severity": 0
//   },
//   "agent_analysis": {
//     "steps_explained_by_agent": [],
//     "correctness_of_guidance": true,
//     "missed_opportunities": [],
//     "agent_sentiment": "",
//     "compliance_flags": []
//   },
//   "kpi": {
//     "customer_talk_ratio": 0.0,
//     "agent_talk_ratio": 0.0,
//     "silence_seconds": 0,
//     "interruption_count": 0,
//     "frustration_score": 0.0,
//     "confusion_level": 0.0
//   },
//   "should_have_done": {
//     "ideal_resolution_path": "",
//     "recommended_followup": "",
//     "department_owner": ""
//   },
//   "dataset_insights": {
//     "similar_calls_count": 0,
//     "city_trend": "",
//     "vintage_trend": "",
//     "probable_root_cause": ""
//   }
// }

// DATASET SUMMARY:
// %s

// TRANSCRIPT:
// \"\"\"
// %s
// \"\"\"

// Notes:
// - Use dataset numbers when producing dataset_insights fields.
// - Evidence lines may be included inside dataset_insights text fields.
// - Ratings: severity 1 (low) to 5 (critical).
// Return only JSON (no commentary).`, string(dsJSON), transcript)

// 	return p
// }

// func ExtractAdvanced(prompt string) (types.KPIExtraction, error) {
// 	log := logger.New().WithField("component", "extractor-advanced")
// 	// mock
// 	if os.Getenv("USE_MOCK_LLM") == "true" {
// 		log.Info("mock LLM mode ON - returning deterministic KPIExtraction")
// 		var mock types.KPIExtraction
// 		mock.CustomerProblem.PrimaryIssue = "pricing"
// 		mock.CustomerProblem.IssueDescription = "Customer confused by GST and final price."
// 		mock.CustomerProblem.UrgencyLevel = "medium"
// 		mock.CustomerProblem.Severity = 3

// 		mock.AgentAnalysis.StepsExplainedByAgent = []string{"Explained listing fees", "Asked customer to retry checkout"}
// 		mock.AgentAnalysis.CorrectnessOfGuidance = false
// 		mock.AgentAnalysis.MissedOpportunities = []string{"Did not offer immediate refund or voucher", "Did not explain GST break-up"}
// 		mock.AgentAnalysis.AgentSentiment = "neutral"
// 		mock.AgentAnalysis.ComplianceFlags = []string{}

// 		mock.KPI.CustomerTalkRatio = 0.62
// 		mock.KPI.AgentTalkRatio = 0.38
// 		mock.KPI.SilenceSeconds = 2
// 		mock.KPI.InterruptionCount = 1
// 		mock.KPI.FrustrationScore = 0.68
// 		mock.KPI.ConfusionLevel = 0.75

// 		mock.ShouldHaveDone.IdealResolutionPath = "Explain GST + offer 1-click refund or voucher; follow-up by onboarding team."
// 		mock.ShouldHaveDone.RecommendedFollowUp = "Call the seller within 24h to assist listing."
// 		mock.ShouldHaveDone.DepartmentOwner = "onboarding"

// 		mock.DatasetInsights.SimilarCallsCount = 1240
// 		mock.DatasetInsights.CityTrend = "Surat: high GST complaints"
// 		mock.DatasetInsights.VintageTrend = "0-2M sellers: highest confusion (67%)"
// 		mock.DatasetInsights.ProbableRootCause = "Sellers unaware of mandatory GST and fee display at checkout."

// 		return mock, nil
// 	}

// 	apiURL := os.Getenv("LLM_GATEWAY_URL")
// 	apiKey := os.Getenv("LLM_API_KEY")
// 	model := os.Getenv("LLM_MODEL")
// 	if apiURL == "" || apiKey == "" {
// 		log.WithFields(map[string]interface{}{
// 			"api_url_set": apiURL != "",
// 			"api_key_set": apiKey != "",
// 		}).Error("llm gateway not configured")
// 		return types.KPIExtraction{}, fmt.Errorf("llm gateway not configured")
// 	}

// 	reqBody := map[string]interface{}{
// 		"model": model,
// 		"messages": []map[string]string{
// 			{"role": "user", "content": prompt},
// 		},
// 		"temperature": 0.0,
// 	}
// 	data, _ := json.MarshalIndent(reqBody, "", "  ")

// 	log.Debug("LLM request payload (full):\n" + string(data))

// 	var lastErr error
// 	var extracted types.KPIExtraction

// 	op := func() error {
// 		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
// 		defer cancel()
// 		req, _ := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(data))
// 		req.Header.Set("Content-Type", "application/json")
// 		req.Header.Set("Authorization", "Bearer "+apiKey)
// 		client := &http.Client{Timeout: 25 * time.Second}
// 		resp, err := client.Do(req)
// 		if err != nil {
// 			lastErr = err
// 			log.WithError(err).Warn("llm request failed")
// 			return err
// 		}
// 		defer resp.Body.Close()
// 		body, _ := io.ReadAll(resp.Body)
// 		log.WithField("http_status", resp.StatusCode).Debug("llm raw response (full):\n" + string(body))
// 		if resp.StatusCode >= 500 {
// 			lastErr = fmt.Errorf("llm server error: %s", string(body))
// 			return lastErr
// 		}

// 		// Attempt to extract JSON substring (robust: remove fences and find balanced JSON)
// 		// content := extractJSON(string(body))
// 		// if content != "" {
// 		// 	log.Debug("extracted JSON payload from LLM (full):\n" + content)
// 		// 	if err := json.Unmarshal([]byte(content), &extracted); err == nil {
// 		// 		lastErr = nil
// 		// 		return nil
// 		// 	} else {
// 		// 		log.WithError(err).Warn("json unmarshal from extracted content failed")
// 		// 		// continue to try other heuristics below (choices / message.content)
// 		// 	}
// 		// } else {
// 		// 	log.Debug("no JSON substring found in raw body")
// 		// }

// 		// Try common "choices" structure (OpenAI/compatible)
// 		var respObj map[string]interface{}
// 		if err := json.Unmarshal(body, &respObj); err == nil {
// 			if choices, ok := respObj["choices"].([]interface{}); ok && len(choices) > 0 {
// 				if c0, ok := choices[0].(map[string]interface{}); ok {
// 					// openai-style: choices[0].message.content
// 					if message, ok := c0["message"].(map[string]interface{}); ok {
// 						if contentRaw, ok := message["content"].(string); ok {
// 							j := extractJSON(contentRaw)
// 							if j != "" {
// 								if err := json.Unmarshal([]byte(j), &extracted); err == nil {
// 									lastErr = nil
// 									return nil
// 								} else {
// 									log.WithError(err).Warn("json unmarshal from choices[0].message.content failed")
// 								}
// 							}
// 						}
// 					}
// 					// or completion.text
// 					if text, ok := c0["text"].(string); ok {
// 						if j := extractJSON(text); j != "" {
// 							if err := json.Unmarshal([]byte(j), &extracted); err == nil {
// 								lastErr = nil
// 								return nil
// 							}
// 						}
// 					}
// 				}
// 			}
// 		}

// 		lastErr = fmt.Errorf("unexpected LLM response (no parsable JSON)")
// 		return lastErr
// 	}

// 	b := backoff.NewExponentialBackOff()
// 	b.MaxElapsedTime = 45 * time.Second
// 	if err := backoff.Retry(op, b); err != nil {
// 		log.WithError(lastErr).Error("llm extract failed after retries")
// 		return types.KPIExtraction{}, fmt.Errorf("llm extract failed: %w", lastErr)
// 	}

// 	log.WithField("parsed_kpi", fmt.Sprintf("%+v", extracted)).Info("parsed KPIExtraction")
// 	return extracted, nil
// }

// // extractJSON attempts to robustly find the first balanced JSON object in a larger string.
// // It strips common markdown fences and then finds the first '{' and its matching closing '}'.
// func extractJSON(s string) string {
// 	if s == "" {
// 		return ""
// 	}

// 	// Normalize line endings
// 	s = strings.ReplaceAll(s, "\r\n", "\n")

// 	// Remove common markdown fences and code markers
// 	// Replace them with empty strings so that JSON braces remain
// 	repls := []string{"```json", "```", "`json", "`", "```yaml", "```text"}
// 	for _, r := range repls {
// 		s = strings.ReplaceAll(s, r, "")
// 	}

// 	// Some LLMs prefix lines like "```json\n{...}\n```". After removing fences above,
// 	// we may still have surrounding explanatory text. Find first '{'.
// 	start := strings.Index(s, "{")
// 	if start == -1 {
// 		return ""
// 	}

// 	// Walk forward, count braces to find matching end
// 	depth := 0
// 	end := -1
// 	for i := start; i < len(s); i++ {
// 		ch := s[i]
// 		if ch == '{' {
// 			depth++
// 		} else if ch == '}' {
// 			depth--
// 			if depth == 0 {
// 				end = i
// 				break
// 			}
// 		}
// 	}
// 	if end == -1 {
// 		// fallback to lastIndex if balanced not found
// 		last := strings.LastIndex(s, "}")
// 		if last > start {
// 			return strings.TrimSpace(s[start : last+1])
// 		}
// 		return ""
// 	}

// 	candidate := strings.TrimSpace(s[start : end+1])

// 	// Final sanity: remove any leading/trailing non-JSON bytes that sneaked in
// 	// (for example weird unicode markers) - try to unmarshal a test
// 	var tmp map[string]interface{}
// 	if err := json.Unmarshal([]byte(candidate), &tmp); err == nil {
// 		return candidate
// 	}

// 	// If unmarshalling failed, try simple cleanup: remove trailing/backslash escapes
// 	candidate = strings.Trim(candidate, "\n\r\t ")
// 	// last attempt
// 	if err := json.Unmarshal([]byte(candidate), &tmp); err == nil {
// 		return candidate
// 	}

// 	// give up and return empty
// 	return ""
// }





























package extractor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"voice-insights-go/internal/logger"
	"voice-insights-go/internal/types"
	"voice-insights-go/internal/dataset"
)

func BuildAdvancedPrompt(transcript string, ds dataset.DatasetSummary) string {
	dsJSON, _ := json.MarshalIndent(ds, "", "  ")
	return fmt.Sprintf(`You are a senior call QA and insights engine. Use the dataset summary as ground truth.

Schema:
{
  "customer_problem": {
    "primary_issue": "",
    "issue_description": "",
    "urgency_level": "", 
    "severity": 0
  },
  "agent_analysis": {
    "steps_explained_by_agent": [],
    "correctness_of_guidance": true,
    "missed_opportunities": [],
    "agent_sentiment": "",
    "compliance_flags": []
  },
  "kpi": {
    "customer_talk_ratio": 0.0,
    "agent_talk_ratio": 0.0,
    "silence_seconds": 0,
    "interruption_count": 0,
    "frustration_score": 0.0,
    "confusion_level": 0.0
  },
  "should_have_done": {
    "ideal_resolution_path": "",
    "recommended_followup": "",
    "department_owner": ""
  },
  "dataset_insights": {
    "similar_calls_count": 0,
    "city_trend": "",
    "vintage_trend": "",
    "probable_root_cause": ""
  }
}

DATASET SUMMARY:
%s

TRANSCRIPT:
"""
%s
"""
Return ONLY JSON.`, string(dsJSON), transcript)
}

func ExtractAdvanced(prompt string) (types.KPIExtraction, error) {
	log := logger.New().WithField("component", "extractor-advanced")

	// mock code unchanged...
	// mock
	if os.Getenv("USE_MOCK_LLM") == "true" {
		log.Info("mock LLM mode ON - returning deterministic KPIExtraction")

		mock := types.KPIExtraction{
			CustomerProblem: types.CustomerProblem{
				PrimaryIssue:     "Fake and irrelevant inquiries",
				IssueDescription: "Customer receives irrelevant and fake leads; agent follow-up missing.",
				UrgencyLevel:     "High",
				Severity:         4,
			},

			AgentAnalysis: types.AgentAnalysis{
				StepsExplainedByAgent: []string{
					"Acknowledged customer frustration",
					"Explained missed-call ticketing",
					"Requested customer to message if calls not reachable",
				},
				CorrectnessOfGuidance: false,
				MissedOpportunities: []string{
					"Did not investigate fake inquiries",
					"Ignored missed 300kg Guwahati inquiry",
				},
				AgentSentiment:  "Neutral / defensive",
				ComplianceFlags: []string{"Lack of proactive ownership"},
			},

			KPI: types.KPIFields{
				CustomerTalkRatio: 0.55,
				AgentTalkRatio:    0.45,
				SilenceSeconds:    12,
				InterruptionCount: 2,
				FrustrationScore:  0.62,
				ConfusionLevel:    0.40,
			},

			ShouldHaveDone: types.ShouldHaveDone{
				IdealResolutionPath: "Investigate missed inquiry, improve lead filtering, explain fake-lead causes.",
				RecommendedFollowUp: "Call customer within 24 hours with resolution plan.",
				DepartmentOwner:     "Sales / Lead Quality Team",
			},

			DatasetInsights: types.DatasetInsights{
				SimilarCallsCount:  42,
				CityTrend:          "Noida shows increasing complaints about irrelevant leads.",
				VintageTrend:       "0–3 month sellers show higher confusion rates.",
				ProbableRootCause:  "Incorrect profile keywords + weak lead filtering",
			},
		}

		return mock, nil
	}

	apiURL := os.Getenv("LLM_GATEWAY_URL")
	apiKey := os.Getenv("LLM_API_KEY")
	model := os.Getenv("LLM_MODEL")
	if apiURL == "" || apiKey == "" {
		return types.KPIExtraction{}, fmt.Errorf("llm gateway not configured")
	}

	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.0,
	}
	data, _ := json.MarshalIndent(reqBody, "", "  ")

	log.Debug("LLM request payload:\n" + string(data))

	var extracted types.KPIExtraction
	var lastErr error

	op := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		req, _ := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(data))
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 25 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			log.WithError(err).Warn("llm request failed")
			return err
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		log.WithField("http_status", resp.StatusCode).Debug("llm raw:\n" + string(body))

		// --- FIX 1: Always try extracting from choices[0].message.content ---
		if inner := extractContentFromChoices(body); inner != "" {
			log.Debug("LLM inner content extracted:\n" + inner)

			if err := json.Unmarshal([]byte(inner), &extracted); err == nil {
				lastErr = nil
				return nil
			}
			log.WithError(err).Warn("Failed to unmarshal choices[0].message.content JSON")
		}

		// --- FIX 2: fallback - traditional extractJSON ---
		if fallback := extractJSON(string(body)); fallback != "" {
			if err := json.Unmarshal([]byte(fallback), &extracted); err == nil {
				lastErr = nil
				return nil
			}
		}

		lastErr = fmt.Errorf("no JSON found in LLM output")
		return lastErr
	}

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 45 * time.Second

	if err := backoff.Retry(op, b); err != nil {
		return types.KPIExtraction{}, fmt.Errorf("llm extract failed: %w", lastErr)
	}

	log.WithField("parsed_kpi", fmt.Sprintf("%+v", extracted)).Info("parsed KPIExtraction")
	return extracted, nil
}

// NEW FUNCTION: Correct extraction from choices[0].message.content
func extractContentFromChoices(body []byte) string {
	var obj map[string]interface{}
	if err := json.Unmarshal(body, &obj); err != nil {
		return ""
	}

	choices, ok := obj["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return ""
	}

	c0, ok := choices[0].(map[string]interface{})
	if !ok {
		return ""
	}

	msg, ok := c0["message"].(map[string]interface{})
	if !ok {
		return ""
	}

	content, ok := msg["content"].(string)
	if !ok {
		return ""
	}

	// Extract JSON inside content
	return extractJSON(content)
}

// OLD extractJSON — unchanged except used properly
func extractJSON(s string) string {
	if s == "" {
		return ""
	}

	s = strings.ReplaceAll(s, "\r\n", "\n")
	repls := []string{"```json", "```", "`json", "`", "```yaml", "```text"}

	for _, r := range repls {
		s = strings.ReplaceAll(s, r, "")
	}

	start := strings.Index(s, "{")
	if start == -1 {
		return ""
	}

	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(s[start : i+1])
			}
		}
	}

	return ""
}
