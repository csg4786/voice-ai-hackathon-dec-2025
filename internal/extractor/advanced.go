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
)

// BuildAdvancedPrompt builds the v2 prompt using the provided transcript and raw search results.
// searchResults can be any JSON-marshallable type (array or object returned by your /search API).
func BuildAdvancedPrompt(transcript string, searchResults any) string {
	srJSON, _ := json.MarshalIndent(searchResults, "", "  ")

	// Strict Schema v2 prompt (return only JSON)
	prompt := `You are an expert Call Quality, Customer Insights, and Resolution Intelligence engine.

Your job is to analyze:
1. The CURRENT CALL TRANSCRIPT
2. The TOP-K MOST RELEVANT HISTORICAL CALLS (SEARCH RESULTS)

Using BOTH inputs, you must produce insights strictly following the JSON schema below.
Your answers MUST be grounded in:
- The transcript
- The provided search results
- NO outside knowledge
- NO hallucinated numbers

If information is missing, leave fields empty or set them to 0/false instead of inventing details.

----------------------------------------------------------------------
SCHEMA v2.0 (STRICT — RETURN ONLY JSON)
{
  "customer_problem": {
    "primary_issue": "",
    "issue_description": "",
    "urgency_level": "",
    "severity": 0,
    "customer_intent": "",
    "repeat_issue": false,
    "related_issue_category": ""
  },

  "agent_analysis": {
    "steps_explained_by_agent": [],
    "correctness_of_guidance": true,
    "missed_opportunities": [],
    "agent_sentiment": "",
    "compliance_flags": [],
    "rapport_score": 0.0,
    "professionalism_score": 0.0,
    "solution_accuracy_score": 0.0,
    "agent_confidence_level": ""
  },

  "kpi": {
    "customer_talk_ratio": 0.0,
    "agent_talk_ratio": 0.0,
    "silence_seconds": 0,
    "interruption_count": 0,
    "frustration_score": 0.0,
    "confusion_level": 0.0,
    "empathy_score": 0.0,
    "resolution_likelihood": 0.0,
    "avg_sentence_length_customer": 0.0,
    "avg_sentence_length_agent": 0.0,
    "dead_air_instances": 0,
    "topic_switch_count": 0
  },

  "should_have_done": {
    "ideal_resolution_path": "",
    "recommended_followup": "",
    "department_owner": "",
    "crucial_missed_questions": [],
    "required_data_points_not_collected": []
  },

  "actions": {
    "executive_actions_required": [],
    "customer_actions_required": [],
    "system_actions_required": [],
    "priority": "",
    "requires_escalation": false,
    "escalation_reason": ""
  },

  "conversation_quality": {
    "overall_score": 0.0,
    "clarity_score": 0.0,
    "listening_score": 0.0,
    "relevance_score": 0.0,
    "trust_building_score": 0.0,
    "red_flags": []
  },

  "trend_insights_from_similar_calls": {
    "similar_calls_count": 0,
    "dominant_issue_category": "",
    "city_trend": "",
    "vintage_trend": "",
    "engagement_level_trend": "",
    "actionability_pattern": "",
    "probable_root_cause": "",
    "recommended_playbook": "",
    "historical_resolution_rate": 0.0,
    "historical_escalation_rate": 0.0
  },

  "business_impact": {
    "risk_of_churn": 0.0,
    "revenue_opportunity_loss": "",
    "customer_ltv_bucket": "",
    "service_gap_identified": "",
    "fix_urgency_level": ""
  }
}
----------------------------------------------------------------------

GUIDELINES FOR ANALYSIS:

1. Ground insights in the transcript.
2. Use SEARCH RESULTS ONLY for:
   - trend patterns
   - common root causes
   - issue repetition
   - city-wise or vintage-wise trends
   - escalation probability
   - resolution playbook inference
   - historical resolution or dissatisfaction patterns

3. DO NOT hallucinate numbers or percentages.
   If unsure, provide qualitative analysis or 0/empty.

4. KPIs must be realistic and derived from:
   - talk ratios
   - interruptions
   - emotional cues
   - agent behavior
   - customer sentiment

5. DO NOT mention the transcript or search results in the final JSON.
   DO NOT include commentary.
   DO NOT escape or wrap JSON in backticks.

----------------------------------------------------------------------
SEARCH RESULTS (Top-K similar calls):
%s

TRANSCRIPT:
%s

----------------------------------------------------------------------
Return ONLY valid JSON that exactly matches SCHEMA v2.0.
`

	return fmt.Sprintf(prompt, string(srJSON), transcript)
}

// FetchSearchResults calls your /search API and returns unmarshalled JSON (any).
func FetchSearchResults(searchAPIURL string, transcript string, k int, httpTimeout time.Duration) (any, error) {
	log := logger.New().WithField("component", "search-client")

	if searchAPIURL == "" {
		return nil, fmt.Errorf("SEARCH_API_URL not configured")
	}

	reqPayload := map[string]any{
		"k":          k,
		"transcript": transcript,
	}
	reqBytes, _ := json.Marshal(reqPayload)

	req, _ := http.NewRequest("POST", searchAPIURL, bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Error("search API request failed")
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Debug("search API raw response:\n" + string(body))

	var parsed any
	if err := json.Unmarshal(body, &parsed); err != nil {
		log.WithError(err).Error("failed to parse search API JSON")
		return nil, err
	}

	return parsed, nil
}

// ExtractAdvanced orchestrates search -> prompt build -> LLM -> parse
// Keeps the same return signature types.KPIExtraction for compatibility.
func ExtractAdvanced(transcript string) (types.KPIExtraction, error) {

	var (
		httpTimeout     = 25 * time.Second
		maxRetryTime    = 45 * time.Second
		searchAPIURL    = os.Getenv("SEARCH_API_URL")
		llmGatewayURL   = os.Getenv("LLM_GATEWAY_URL")
		llmGatewayModel = os.Getenv("LLM_MODEL")
		llmAPIKey       = os.Getenv("LLM_API_KEY")
	)

	log := logger.New().WithField("component", "extractor-advanced")

	// mock
	if os.Getenv("USE_MOCK_LLM") == "true" {
		log.Info("mock LLM mode ON - returning deterministic KPIExtraction")
		// Keep backward-compatible mock (you can update this if you extend types)
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
					"Explained lead filtering",
					"Requested customer to send spam numbers",
				},
				CorrectnessOfGuidance: false,
				MissedOpportunities: []string{
					"Did not analyze lead patterns",
					"Did not recommend category cleanup",
				},
				AgentSentiment:         "Neutral",
				ComplianceFlags:        []string{"Lack of ownership"},
				RapportScore:           0.6,
				ProfessionalismScore:   0.7,
				SolutionAccuracyScore:  0.4,
				AgentConfidenceLevel:   "Medium",
			},
			KPI: types.KPIFields{
				CustomerTalkRatio:        0.55,
				AgentTalkRatio:           0.45,
				SilenceSeconds:           12,
				InterruptionCount:        2,
				FrustrationScore:         0.62,
				ConfusionLevel:           0.40,
				EmpathyScore:             0.50,
				ResolutionLikelihood:     0.30,
				AvgSentenceLengthCustomer: 6.2,
				AvgSentenceLengthAgent:    5.1,
				DeadAirInstances:          1,
				TopicSwitchCount:          4,
			},
			ShouldHaveDone: types.ShouldHaveDone{
				IdealResolutionPath: "Investigate lead source, fix profile keywords, reduce irrelevant leads.",
				RecommendedFollowUp: "Call customer after fixing settings.",
				DepartmentOwner:     "Lead Quality Team",
				CrucialMissedQuestions: []string{
					"What product categories do you NOT deal in?",
				},
				RequiredDataPointsNotCollected: []string{
					"Customer's minimum quantity threshold",
				},
			},
			Actions: types.Actions{
				ExecutiveActionsRequired: []string{
					"Check category mapping",
					"Add missing steel keywords",
				},
				CustomerActionsRequired: []string{
					"Share spam callers",
				},
				SystemActionsRequired: []string{
					"Improve lead filtering model",
				},
				Priority:          "High",
				RequiresEscalation: false,
				EscalationReason:  "",
			},
			ConversationQuality: types.ConversationQuality{
				OverallScore:       0.55,
				ClarityScore:       0.60,
				ListeningScore:     0.70,
				RelevanceScore:     0.50,
				TrustBuildingScore: 0.40,
				RedFlags:           []string{"Agent defensive tone"},
			},
			TrendInsights: types.TrendInsights{
				SimilarCallsCount:        37,
				DominantIssueCategory:    "Lead Quality",
				CityTrend:                "High lead dissatisfaction in similar tier-2 cities",
				VintageTrend:             "Mid-vintage sellers complain about spam",
				EngagementLevelTrend:     "High-engagement sellers more vocal about quality",
				ActionabilityPattern:     "Profile cleanup + keyword correction fix ~70% of issues",
				ProbableRootCause:        "Incorrect keywords + spam call leakage",
				RecommendedPlaybook:      "Lead Quality SOP v3",
				HistoricalResolutionRate: 0.72,
				HistoricalEscalationRate: 0.18,
			},
			BusinessImpact: types.BusinessImpact{
				RiskOfChurn:           0.35,
				RevenueOpportunityLoss: "Medium",
				CustomerLTVBucket:     "Medium",
				ServiceGapIdentified:  "Lead filtering performance",
				FixUrgencyLevel:       "High",
			},
		}
		return mock, nil
	}

	// 1) call search API (k=3)
	searchResults, err := FetchSearchResults(searchAPIURL, transcript, 3, httpTimeout)
	if err != nil {
		return types.KPIExtraction{}, fmt.Errorf("search API failed: %w", err)
	}

	// 2) build prompt using search results + transcript
	prompt := BuildAdvancedPrompt(transcript, searchResults)

	// 3) prepare LLM request payload
	if llmGatewayURL == "" || llmAPIKey == "" {
		return types.KPIExtraction{}, fmt.Errorf("llm gateway not configured")
	}
	reqBody := map[string]any{
		"model": llmGatewayModel,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.0,
	}
	data, _ := json.MarshalIndent(reqBody, "", "  ")
	log.Debug("LLM request payload (sizes):", "payload_len", len(data))

	var extracted types.KPIExtraction
	var lastErr error

	// LLM call with retry/backoff
	op := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
		defer cancel()

		req, _ := http.NewRequestWithContext(ctx, "POST", llmGatewayURL, bytes.NewReader(data))
		req.Header.Set("Authorization", "Bearer "+llmAPIKey)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: httpTimeout}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			log.WithError(err).Warn("llm request failed")
			return err
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		log.WithField("http_status", resp.StatusCode).Debug("llm raw:\n" + string(body))

		// Try choices[0].message.content (OpenAI-like)
		if inner := extractContentFromChoices(body); inner != "" {
			log.Debug("extracted JSON from choices content")
			if err := json.Unmarshal([]byte(inner), &extracted); err == nil {
				lastErr = nil
				return nil
			}
			log.WithError(err).Warn("unmarshal from choices content failed")
		}

		// Fallback: find first balanced JSON in response body
		if fallback := extractJSON(string(body)); fallback != "" {
			if err := json.Unmarshal([]byte(fallback), &extracted); err == nil {
				lastErr = nil
				return nil
			}
			log.WithError(err).Warn("unmarshal from fallback JSON failed")
		}

		// Couldn't parse; log and return temporary error to retry (unless 4xx)
		lastErr = fmt.Errorf("no JSON found in LLM output")
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			// Permanent: don't retry on client errors
			return backoff.Permanent(lastErr)
		}
		return lastErr
	}

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = maxRetryTime

	if err := backoff.Retry(op, b); err != nil {
		return types.KPIExtraction{}, fmt.Errorf("llm extract failed: %w", lastErr)
	}

	log.WithField("parsed_kpi", fmt.Sprintf("%+v", extracted)).Info("parsed KPIExtraction")
	return extracted, nil
}

// extractContentFromChoices attempts to read openai-style choices[0].message.content JSON
func extractContentFromChoices(body []byte) string {
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return ""
	}

	choices, ok := obj["choices"].([]any)
	if !ok || len(choices) == 0 {
		return ""
	}
	c0, _ := choices[0].(map[string]any)
	if c0 == nil {
		return ""
	}
	msg, _ := c0["message"].(map[string]any)
	if msg == nil {
		return ""
	}
	content, _ := msg["content"].(string)
	return extractJSON(content)
}

// extractJSON finds the first balanced JSON object in a string and returns it.
// It strips common markdown fences first.
func extractJSON(s string) string {
	if s == "" {
		return ""
	}

	// normalize newlines
	s = strings.ReplaceAll(s, "\r\n", "\n")

	// Remove markdown fences (commonly output by LLMs)
	for _, r := range []string{"```json", "```yaml", "```text", "```", "`json", "`"} {
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
				candidate := strings.TrimSpace(s[start : i+1])
				// quick sanity check
				var tmp any
				if json.Unmarshal([]byte(candidate), &tmp) == nil {
					return candidate
				}
				// otherwise return candidate anyway (best effort)
				return candidate
			}
		}
	}

	// no balanced found
	return ""
}
