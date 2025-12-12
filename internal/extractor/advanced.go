// internal/extractor/advanced.go
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

// BuildAdvancedPrompt builds a single prompt string combining the dataset summary and the transcript.
func BuildAdvancedPrompt(transcript string, ds dataset.DatasetSummary) string {
	dsJSON, _ := json.MarshalIndent(ds, "", "  ")
	p := fmt.Sprintf(`You are a senior call QA and insights engine. Use the dataset summary as ground truth (do not invent numbers). Produce ONLY valid JSON matching the schema below.

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
\"\"\"
%s
\"\"\"

Notes:
- Use dataset numbers when producing dataset_insights fields.
- Evidence lines may be included inside dataset_insights text fields.
- Ratings: severity 1 (low) to 5 (critical).
Return only JSON (no commentary).`, string(dsJSON), transcript)

	return p
}

// ExtractAdvanced sends the prompt to LLM gateway and parses the KPIExtraction result.
// It supports USE_MOCK_LLM=true for offline testing.
func ExtractAdvanced(prompt string) (types.KPIExtraction, error) {
	log := logger.New().WithField("component", "extractor-advanced")
	if os.Getenv("USE_MOCK_LLM") == "true" {
		log.Info("mock LLM mode ON - returning deterministic KPIExtraction")
		// deterministic mock
		var mock types.KPIExtraction
		mock.CustomerProblem.PrimaryIssue = "pricing"
		mock.CustomerProblem.IssueDescription = "Customer confused by GST and final price."
		mock.CustomerProblem.UrgencyLevel = "medium"
		mock.CustomerProblem.Severity = 3

		mock.AgentAnalysis.StepsExplainedByAgent = []string{"Explained listing fees", "Asked customer to retry checkout"}
		mock.AgentAnalysis.CorrectnessOfGuidance = false
		mock.AgentAnalysis.MissedOpportunities = []string{"Did not offer immediate refund or voucher", "Did not explain GST break-up"}
		mock.AgentAnalysis.AgentSentiment = "neutral"
		mock.AgentAnalysis.ComplianceFlags = []string{}

		mock.KPI.CustomerTalkRatio = 0.62
		mock.KPI.AgentTalkRatio = 0.38
		mock.KPI.SilenceSeconds = 2
		mock.KPI.InterruptionCount = 1
		mock.KPI.FrustrationScore = 0.68
		mock.KPI.ConfusionLevel = 0.75

		mock.ShouldHaveDone.IdealResolutionPath = "Explain GST + offer 1-click refund or voucher; follow-up by onboarding team."
		mock.ShouldHaveDone.RecommendedFollowUp = "Call the seller within 24h to assist listing."
		mock.ShouldHaveDone.DepartmentOwner = "onboarding"

		mock.DatasetInsights.SimilarCallsCount = 1240
		mock.DatasetInsights.CityTrend = "Surat: high GST complaints"
		mock.DatasetInsights.VintageTrend = "0-2M sellers: highest confusion (67%)"
		mock.DatasetInsights.ProbableRootCause = "Sellers unaware of mandatory GST and fee display at checkout."

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
	data, _ := json.Marshal(reqBody)

	var lastErr error
	var extracted types.KPIExtraction

	op := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		req, _ := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)
		client := &http.Client{Timeout: 20 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			return err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("llm server error: %s", string(body))
			return lastErr
		}
		// Attempt to find the JSON substring, similar to previous robust parsing
		start := bytes.Index(body, []byte("{"))
		end := bytes.LastIndex(body, []byte("}"))
		if start >= 0 && end > start {
			raw := body[start : end+1]
			if err := json.Unmarshal(raw, &extracted); err == nil {
				lastErr = nil
				return nil
			}
		}
		// try to parse a typical choices structure
		var respObj map[string]interface{}
		if err := json.Unmarshal(body, &respObj); err == nil {
			if choices, ok := respObj["choices"].([]interface{}); ok && len(choices) > 0 {
				if c0, ok := choices[0].(map[string]interface{}); ok {
					if message, ok := c0["message"].(map[string]interface{}); ok {
						if content, ok := message["content"].(string); ok {
							s := content
							start := strings.Index(s, "{")
							end := strings.LastIndex(s, "}")
							if start >= 0 && end > start {
								raw := s[start : end+1]
								if err := json.Unmarshal([]byte(raw), &extracted); err == nil {
									lastErr = nil
									return nil
								}
							}
						}
					}
				}
			}
		}
		lastErr = fmt.Errorf("unexpected LLM response: %s", string(body))
		return lastErr
	}

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 30 * time.Second
	if err := backoff.Retry(op, b); err != nil {
		return types.KPIExtraction{}, fmt.Errorf("llm extract failed: %w", lastErr)
	}

	return extracted, nil
}
