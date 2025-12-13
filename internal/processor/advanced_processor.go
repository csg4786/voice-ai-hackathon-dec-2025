package processor

import (
	"fmt"
	"os"
	"time"

	"voice-insights-go/internal/extractor"
	"voice-insights-go/internal/logger"
	"voice-insights-go/internal/transcription"
	"voice-insights-go/internal/types"
)

// ProcessSingleCall advanced flow (returns types.KPIResult)
// NOTE: The DatasetSummary is removed because insights now come purely from k-relevant search.
func ProcessSingleCall(audioURL string, k int, timeout time.Duration) (types.KPIResult, error) {
	log := logger.New().WithField("component", "processor")
	start := time.Now()

	// initialize a safe default KPIResult with v2-complete fields
	res := types.KPIResult{
		AudioURL:   audioURL,
		Transcript: "",
		KPI:        emptyExtractionV2(),
		Evidence:   map[string]interface{}{},
		DurationMs: 0,
		Error:      "",
	}

	log = log.WithField("audio_url", audioURL)
	log.Info("processor start")

	// -------------------------------------------------------------
	// MOCK MODE — skip transcription + LLM
	// -------------------------------------------------------------
	if os.Getenv("USE_MOCK_LLM") == "true" {
		log.Warn("MOCK MODE ENABLED — returning synthetic KPIResult")

		res.Transcript = `Speaker 1: Hello, I need help.
Speaker 2: Sure sir, let me check your account.
Speaker 1: I am receiving fake enquiries.
Speaker 2: I understand sir, I will investigate.`

		res.KPI = mockExtractionV2()
		res.Evidence = map[string]interface{}{
			"mode": "mock",
			"reason": "USE_MOCK_LLM=true",
		}

		res.DurationMs = time.Since(start).Milliseconds()
		log.Info("MOCK PROCESSOR: returning synthetic KPIResult")
		return res, nil
	}

	// -------------------------------------------------------------
	// STEP 1 — TRANSCRIPTION
	// -------------------------------------------------------------
	tr, err := transcription.GetTranscript(audioURL)
	if err != nil {
		res.Error = fmt.Sprintf("transcription error: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		log.WithError(err).Warn("transcription failed")
		return res, err
	}
	res.Transcript = tr
	log.WithField("transcript_len", len(tr)).Info("got transcript")

	// -------------------------------------------------------------
	// STEP 2 — EXTRACTION (search + LLM)
	// -------------------------------------------------------------
	kpiExtract, err := extractor.ExtractAdvanced(tr, k) // No dataset summary, extractor handles search internally
	if err != nil {
		res.Error = fmt.Sprintf("llm extraction error: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		log.WithError(err).Warn("llm extraction failed")
		return res, err
	}

	// ensure nil-slices are not nil
	normalizeExtractionV2(&kpiExtract)

	res.KPI = kpiExtract
	log.WithField("primary_issue", kpiExtract.CustomerProblem.PrimaryIssue).Info("llm extracted KPI")

	// -------------------------------------------------------------
	// STEP 3 — EVIDENCE BLOCK (search-based)
	// -------------------------------------------------------------
	res.Evidence = map[string]interface{}{
		"insight_source":  "k-relevant-search",
		"transcript_chars": len(tr),
		"has_trends":       true,
		"similarity_info": map[string]interface{}{
			"similar_calls_count": kpiExtract.TrendInsights.SimilarCallsCount,
			"dominant_issue":      kpiExtract.TrendInsights.DominantIssueCategory,
			"probable_root_cause": kpiExtract.TrendInsights.ProbableRootCause,
		},
	}

	res.DurationMs = time.Since(start).Milliseconds()
	log.WithFields(map[string]interface{}{
		"duration_ms": res.DurationMs,
	}).Info("processor completed")

	return res, nil
}

/* ------------------------------------------------------------
   HELPERS
------------------------------------------------------------ */

// returns a fully zeroed Schema v2 extraction object
func emptyExtractionV2() types.KPIExtraction {
	return types.KPIExtraction{
		CustomerProblem:      types.CustomerProblem{},
		AgentAnalysis:        types.AgentAnalysis{StepsExplainedByAgent: []string{}, MissedOpportunities: []string{}, ComplianceFlags: []string{}},
		KPI:                  types.KPIFields{},
		ShouldHaveDone:       types.ShouldHaveDone{CrucialMissedQuestions: []string{}, RequiredDataPointsNotCollected: []string{}},
		Actions:              types.Actions{ExecutiveActionsRequired: []string{}, CustomerActionsRequired: []string{}, SystemActionsRequired: []string{}},
		ConversationQuality:  types.ConversationQuality{RedFlags: []string{}},
		TrendInsights:        types.TrendInsights{},
		BusinessImpact:       types.BusinessImpact{},
	}
}

// ensure slices are not nil (makes JSON consistent)
func normalizeExtractionV2(x *types.KPIExtraction) {
	if x.AgentAnalysis.StepsExplainedByAgent == nil {
		x.AgentAnalysis.StepsExplainedByAgent = []string{}
	}
	if x.AgentAnalysis.MissedOpportunities == nil {
		x.AgentAnalysis.MissedOpportunities = []string{}
	}
	if x.AgentAnalysis.ComplianceFlags == nil {
		x.AgentAnalysis.ComplianceFlags = []string{}
	}
	if x.ShouldHaveDone.CrucialMissedQuestions == nil {
		x.ShouldHaveDone.CrucialMissedQuestions = []string{}
	}
	if x.ShouldHaveDone.RequiredDataPointsNotCollected == nil {
		x.ShouldHaveDone.RequiredDataPointsNotCollected = []string{}
	}
	if x.Actions.ExecutiveActionsRequired == nil {
		x.Actions.ExecutiveActionsRequired = []string{}
	}
	if x.Actions.CustomerActionsRequired == nil {
		x.Actions.CustomerActionsRequired = []string{}
	}
	if x.Actions.SystemActionsRequired == nil {
		x.Actions.SystemActionsRequired = []string{}
	}
	if x.ConversationQuality.RedFlags == nil {
		x.ConversationQuality.RedFlags = []string{}
	}
}

// synthetic mock object (schema v2)
func mockExtractionV2() types.KPIExtraction {
	return types.KPIExtraction{
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
}
