// package processor

// import (
// 	"fmt"
// 	"strings"
// 	"time"

// 	"voice-insights-go/internal/dataset"
// 	"voice-insights-go/internal/extractor"
// 	"voice-insights-go/internal/logger"
// 	"voice-insights-go/internal/transcription"
// 	"voice-insights-go/internal/types"
// )

// // KPIResult is returned by /process
// type KPIResult struct {
// 	AudioURL   string                 `json:"audio_url"`
// 	Transcript string                 `json:"transcript"`
// 	KPI        types.KPIExtraction    `json:"kpi_extraction"`
// 	Evidence   map[string]interface{} `json:"evidence"`
// 	DurationMs int64                  `json:"duration_ms"`
// 	Error      string                 `json:"error,omitempty"`
// }

// // ProcessSingleCallWithDataset advanced flow
// func ProcessSingleCallWithDataset(audioURL string, timeout time.Duration, ds dataset.DatasetSummary) (KPIResult, error) {
// 	log := logger.New().WithField("component", "processor").WithField("audio_url", audioURL)
// 	start := time.Now()
// 	res := KPIResult{AudioURL: audioURL}

// 	// Transcription
// 	trStart := time.Now()
// 	tr, err := transcription.GetTranscript(audioURL)
// 	trDur := time.Since(trStart)
// 	if err != nil {
// 		res.Error = fmt.Sprintf("transcription error: %v", err)
// 		res.DurationMs = time.Since(start).Milliseconds()
// 		log.WithError(err).WithField("transcription_ms", trDur.Milliseconds()).Warn("transcription failed")
// 		return res, err
// 	}
// 	res.Transcript = tr
// 	log.WithField("transcription_ms", trDur.Milliseconds()).WithField("transcript_len", len(tr)).Info("transcription success")
// 	// Log transcript full (user requested full logging)
// 	log.Debug("transcript full:\n" + tr)

// 	// Build prompt
// 	prompt := extractor.BuildAdvancedPrompt(tr, ds)
// 	log.WithField("prompt_len", len(prompt)).Info("prompt built for LLM")
// 	log.Debug("prompt full:\n" + prompt)

// 	// LLM extraction
// 	llmStart := time.Now()
// 	kpi, err := extractor.ExtractAdvanced(prompt)
// 	llmDur := time.Since(llmStart)
// 	if err != nil {
// 		res.Error = fmt.Sprintf("llm extraction error: %v", err)
// 		res.DurationMs = time.Since(start).Milliseconds()
// 		log.WithError(err).WithField("llm_ms", llmDur.Milliseconds()).Warn("llm extraction failed")
// 		return res, err
// 	}
// 	res.KPI = kpi
// 	log.WithField("llm_ms", llmDur.Milliseconds()).Info("llm extraction success")

// 	// Dataset grounding: augment dataset_insights with in-memory heuristics
// 	evidence := map[string]interface{}{}
// 	evidence["dataset_total_calls"] = ds.TotalCalls

// 	pi := strings.ToLower(kpi.CustomerProblem.PrimaryIssue)
// 	similar := 0
// 	for cat, cnt := range ds.ByCategory {
// 		if strings.Contains(strings.ToLower(cat), pi) || strings.Contains(pi, strings.ToLower(cat)) {
// 			similar += cnt
// 		}
// 	}
// 	if similar == 0 {
// 		for _, cnt := range ds.ByCategory {
// 			similar += cnt
// 		}
// 	}
// 	evidence["similar_calls_count_estimate"] = similar
// 	log.WithField("similar_estimate", similar).Info("computed similar calls estimate")

// 	// city trend
// 	cityFound := ""
// 	for city := range ds.ByCityTopN {
// 		if strings.Contains(strings.ToLower(tr), strings.ToLower(city)) {
// 			cityFound = city
// 			break
// 		}
// 	}
// 	if cityFound != "" {
// 		evidence["matched_city"] = cityFound
// 		evidence["city_top_issues"] = ds.ByCityTopN[cityFound]
// 		log.WithField("matched_city", cityFound).Info("matched city to transcript")
// 	} else {
// 		evidence["matched_city"] = nil
// 		log.Info("no city matched from transcript")
// 	}

// 	// vintage
// 	if strings.Contains(strings.ToLower(tr), "register") || strings.Contains(strings.ToLower(tr), "just registered") {
// 		evidence["probable_vintage_bucket"] = "0-2"
// 	} else {
// 		evidence["probable_vintage_bucket"] = "unknown"
// 	}
// 	res.Evidence = evidence

// 	res.DurationMs = time.Since(start).Milliseconds()
// 	log.WithField("duration_ms", res.DurationMs).Info("process completed")
// 	return res, nil
// }


package processor

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"voice-insights-go/internal/dataset"
	"voice-insights-go/internal/extractor"
	"voice-insights-go/internal/logger"
	"voice-insights-go/internal/transcription"
	"voice-insights-go/internal/types"
)

// ProcessSingleCallWithDataset advanced flow (returns types.KPIResult)
func ProcessSingleCallWithDataset(audioURL string, timeout time.Duration, ds dataset.DatasetSummary) (types.KPIResult, error) {
	log := logger.New().WithField("component", "processor")
	start := time.Now()

	// initialize result with safe defaults
	res := types.KPIResult{
		AudioURL:   audioURL,
		Transcript: "",
		KPI: types.KPIExtraction{
			CustomerProblem: types.CustomerProblem{},
			AgentAnalysis: types.AgentAnalysis{
				StepsExplainedByAgent: []string{},
				MissedOpportunities:   []string{},
				ComplianceFlags:       []string{},
			},
			KPI: types.KPIFields{},
			ShouldHaveDone: types.ShouldHaveDone{},
			DatasetInsights: types.DatasetInsights{},
		},
		Evidence:   map[string]interface{}{},
		DurationMs: 0,
		Error:      "",
	}

	log = log.WithField("audio_url", audioURL)
	log.Info("processor start")

		// -------------------------------------------------------------
	// MOCK MODE: If USE_MOCK_LLM=true, skip transcription + LLM
	// -------------------------------------------------------------
	if os.Getenv("USE_MOCK_LLM") == "true" {
		log.Warn("MOCK MODE ENABLED — returning synthetic KPIResult")

		res.Transcript = `Speaker 1: Hello, I need help.
	Speaker 2: Sure sir, let me check your account.
	Speaker 1: I am receiving fake enquiries.
	Speaker 2: I understand sir, I will investigate.`

		// Fill mock KPIExtraction exactly like the LLM mock version
		res.KPI = types.KPIExtraction{
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

		// Evidence block consistent with real processor behavior
		res.Evidence = map[string]interface{}{
			"dataset_total_calls":        ds.TotalCalls,
			"similar_calls_count_estimate": 42,
			"matched_city":              "noida",
			"city_top_issues":           []string{"register", "payment"},
			"probable_vintage_bucket":   "unknown",
		}

		res.DurationMs = time.Since(start).Milliseconds()

		log.Info("MOCK PROCESSOR: returning synthetic KPIResult")
		return res, nil
	}


	// --- transcription ---
	tr, err := transcription.GetTranscript(audioURL)
	if err != nil {
		res.Error = fmt.Sprintf("transcription error: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		log.WithError(err).Warn("transcription failed")
		return res, err
	}
	res.Transcript = tr
	log.WithField("transcript_len", len(tr)).Info("got transcript")

	// --- build prompt & extract via LLM ---
	prompt := extractor.BuildAdvancedPrompt(tr, ds)
	kpiExtract, err := extractor.ExtractAdvanced(prompt)
	if err != nil {
		res.Error = fmt.Sprintf("llm extraction error: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		log.WithError(err).Warn("llm extraction failed")
		return res, err
	}

	// Ensure the extracted object has safe arrays (no nil slices)
	if kpiExtract.AgentAnalysis.StepsExplainedByAgent == nil {
		kpiExtract.AgentAnalysis.StepsExplainedByAgent = []string{}
	}
	if kpiExtract.AgentAnalysis.MissedOpportunities == nil {
		kpiExtract.AgentAnalysis.MissedOpportunities = []string{}
	}
	if kpiExtract.AgentAnalysis.ComplianceFlags == nil {
		kpiExtract.AgentAnalysis.ComplianceFlags = []string{}
	}

	res.KPI = kpiExtract
	log.WithField("kpi_summary", kpiExtract.CustomerProblem.PrimaryIssue).Info("llm extracted KPI")

	// --- dataset grounding / evidence augmentation ---
	evidence := map[string]interface{}{}
	evidence["dataset_total_calls"] = ds.TotalCalls

	// simple similar calls estimation by category matching
	pi := strings.ToLower(kpiExtract.CustomerProblem.PrimaryIssue)
	similar := 0
	if pi != "" {
		for cat, cnt := range ds.ByCategory {
			if strings.Contains(strings.ToLower(cat), pi) || strings.Contains(pi, strings.ToLower(cat)) {
				similar += cnt
			}
		}
	}
	if similar == 0 {
		// fallback: use overall total as conservative estimate
		similar = ds.TotalCalls
	}
	evidence["similar_calls_count_estimate"] = similar

	// try to find city mention in transcript
	matchedCity := ""
	transLower := strings.ToLower(tr)

	for city := range ds.ByCityTopN {
		cityLower := strings.ToLower(city)

		// create safe regex for exact word match
		pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(cityLower))

		if ok, _ := regexp.MatchString(pattern, transLower); ok {
			matchedCity = city
			break
		}
	}

	if matchedCity != "" {
		evidence["matched_city"] = matchedCity
		evidence["city_top_issues"] = ds.ByCityTopN[matchedCity]
	} else {
		evidence["matched_city"] = nil
	}

	// vintage bucket guess
	if strings.Contains(strings.ToLower(tr), "register") || strings.Contains(strings.ToLower(tr), "just registered") {
		evidence["probable_vintage_bucket"] = "0-2"
	} else {
		evidence["probable_vintage_bucket"] = "unknown"
	}

	res.Evidence = evidence
	res.DurationMs = time.Since(start).Milliseconds()
	log.WithFields(map[string]interface{}{
		"duration_ms": res.DurationMs,
		"similar_est": similar,
	}).Info("processor completed")

	return res, nil
}
