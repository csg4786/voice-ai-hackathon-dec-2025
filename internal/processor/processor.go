package processor

// import (
// 	"encoding/json"
// 	"fmt"
// 	"time"

// 	"voice-insights-go/internal/actionable"
// 	"voice-insights-go/internal/aggregator"
// 	"voice-insights-go/internal/dataset"
// 	"voice-insights-go/internal/extractor"
// 	// "voice-insights-go/internal/logger"
// 	"voice-insights-go/internal/transcription"
// 	"voice-insights-go/internal/types"
// )

// type ProcessResult struct {
// 	AudioURL   string             `json:"audio_url"`
// 	Transcript string             `json:"transcript"`
// 	Extraction types.Extraction   `json:"extraction"`
// 	Evidence   map[string]interface{} `json:"evidence"`
// 	ActionCard actionable.ActionCard `json:"action_card"`
// 	DurationMs int64              `json:"duration_ms"`
// 	Error      string             `json:"error,omitempty"`
// }

// func ProcessSingleCallWithDataset(audioURL string, timeout time.Duration, ds dataset.DatasetSummary) (ProcessResult, error) {
// 	// log := logger.New().WithField("component", "processor")
// 	start := time.Now()
// 	res := ProcessResult{AudioURL: audioURL}
// 	// 1) Transcription
// 	tr, err := transcription.GetTranscript(audioURL)
// 	if err != nil {
// 		res.Error = fmt.Sprintf("transcription error: %v", err)
// 		res.DurationMs = time.Since(start).Milliseconds()
// 		return res, err
// 	}
// 	res.Transcript = tr
// 	// 2) Build prompt with dataset summary and transcript
// 	dsJSON, _ := json.MarshalIndent(ds, "", "  ")
// 	prompt := fmt.Sprintf(`You are an insights engine. Use the following dataset summary as ground truth (do NOT invent dataset facts): %s

// Analyze this call transcript:
// """%s"""

// Return ONLY a JSON object with keys:
// category (one of onboarding, pricing, payment, delivery, product_issue, verification, support, other),
// is_confused (true/false),
// sentiment (positive|neutral|negative),
// escalation_reason (short phrase or empty),
// root_cause (one-sentence),
// evidence (list of dataset facts that support your conclusion),
// next_best_action (one-line actionable instruction: owner + urgency).
// `, string(dsJSON), tr)
// 	// 3) LLM extraction
// 	extr, err := extractor.ExtractWithPrompt(prompt)
// 	if err != nil {
// 		res.Error = fmt.Sprintf("llm extraction error: %v", err)
// 		res.DurationMs = time.Since(start).Milliseconds()
// 		return res, err
// 	}
// 	res.Extraction = extr
// 	// 4) Evidence object
// 	res.Evidence = map[string]interface{}{
// 		"dataset_total_calls": ds.TotalCalls,
// 		"dataset_by_category": ds.ByCategory,
// 		"top_examples":        ds.TopExampleTranscripts,
// 	}
// 	// 5) action card via aggregator+actionable
// 	enr := types.EnrichedRecord{
// 		CallRecord: types.CallRecord{AudioURL: audioURL, Transcript: tr},
// 		Extraction: extr,
// 		VintageBucket: "single-call",
// 	}
// 	ag := aggregator.Aggregate([]types.EnrichedRecord{enr})
// 	res.ActionCard = actionable.Generate(ag)
// 	res.DurationMs = time.Since(start).Milliseconds()
// 	return res, nil
// }
