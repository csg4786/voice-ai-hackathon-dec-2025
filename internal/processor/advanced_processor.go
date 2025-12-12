// internal/processor/advanced_processor.go
package processor

import (
	"fmt"
	"strings"
	"time"

	"voice-insights-go/internal/dataset"
	"voice-insights-go/internal/extractor"
	// "voice-insights-go/internal/logger"
	"voice-insights-go/internal/transcription"
	"voice-insights-go/internal/types"
)

// KPIResult is returned by /process
type KPIResult struct {
	AudioURL   string               `json:"audio_url"`
	Transcript string               `json:"transcript"`
	KPI        types.KPIExtraction  `json:"kpi_extraction"`
	Evidence   map[string]interface{} `json:"evidence"`
	DurationMs int64                `json:"duration_ms"`
	Error      string               `json:"error,omitempty"`
}

// ProcessSingleCallWithDataset advanced flow
func ProcessSingleCallWithDataset(audioURL string, timeout time.Duration, ds dataset.DatasetSummary) (KPIResult, error) {
	// log := logger.New().WithField("component", "advanced-processor")
	start := time.Now()
	res := KPIResult{AudioURL: audioURL}

	// Transcription (mockable)
	tr, err := transcription.GetTranscript(audioURL)
	if err != nil {
		res.Error = fmt.Sprintf("transcription error: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res, err
	}
	res.Transcript = tr

	// Build prompt
	prompt := extractor.BuildAdvancedPrompt(tr, ds)

	// LLM extraction (mockable)
	kpi, err := extractor.ExtractAdvanced(prompt)
	if err != nil {
		res.Error = fmt.Sprintf("llm extraction error: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res, err
	}
	res.KPI = kpi

	// Dataset grounding: try to augment dataset_insights with simple in-memory searches
	evidence := map[string]interface{}{}
	evidence["dataset_total_calls"] = ds.TotalCalls
	// approximate similar calls by keyword match on primary_issue
	pi := strings.ToLower(kpi.CustomerProblem.PrimaryIssue)
	similar := 0
	for cat, cnt := range ds.ByCategory {
		if strings.Contains(strings.ToLower(cat), pi) || strings.Contains(pi, strings.ToLower(cat)) {
			similar += cnt
		}
	}
	if similar == 0 {
		// fallback: sum top categories if primary_issue == "other"
		for _, cnt := range ds.ByCategory {
			similar += cnt
		}
	}
	evidence["similar_calls_count_estimate"] = similar

	// city trend: attempt to match mention of city in transcript to dataset summary
	cityFound := ""
	for city := range ds.ByCityTopN {
		if strings.Contains(strings.ToLower(tr), strings.ToLower(city)) {
			cityFound = city
			break
		}
	}
	if cityFound != "" {
		evidence["matched_city"] = cityFound
		evidence["city_top_issues"] = ds.ByCityTopN[cityFound]
	} else {
		evidence["matched_city"] = nil
	}

	// vintage trend: if transcript includes words like "new", "just registered", map to 0-2
	if strings.Contains(strings.ToLower(tr), "register") || strings.Contains(strings.ToLower(tr), "just registered") || strings.Contains(strings.ToLower(tr), "new seller") {
		evidence["probable_vintage_bucket"] = "0-2"
	} else {
		evidence["probable_vintage_bucket"] = "unknown"
	}
	res.Evidence = evidence

	res.DurationMs = time.Since(start).Milliseconds()
	return res, nil
}
