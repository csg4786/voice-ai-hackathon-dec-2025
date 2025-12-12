// internal/pipeline/pipeline.go
package pipeline

// import (
// 	"context"
// 	"fmt"
// 	"time"

// 	"voice-insights-go/internal/extractor"
// 	"voice-insights-go/internal/transcription"
// 	"voice-insights-go/internal/types"
// )

// // ProcessWithContext runs the pipeline for one call with an overall timeout.
// // returns EnrichedRecord or error.
// func ProcessWithContext(r types.CallRecord, timeout time.Duration) (types.EnrichedRecord, error) {
// 	ctx, cancel := context.WithTimeout(context.Background(), timeout)
// 	defer cancel()

// 	// transcription
// 	type tResult struct {
// 		text string
// 		err  error
// 	}
// 	trCh := make(chan tResult, 1)
// 	go func() {
// 		t, err := transcription.GetTranscript(r.AudioURL)
// 		trCh <- tResult{t, err}
// 	}()

// 	select {
// 	case <-ctx.Done():
// 		return types.EnrichedRecord{}, fmt.Errorf("pipeline timeout during transcription")
// 	case tr := <-trCh:
// 		if tr.err != nil {
// 			return types.EnrichedRecord{}, fmt.Errorf("transcription error: %w", tr.err)
// 		}
// 		r.Transcript = tr.text
// 	}

// 	// LLM extraction (synchronous but with its own internal timeouts/retries)
// 	ext, err := extractor.Extract(r.Transcript)
// 	if err != nil {
// 		return types.EnrichedRecord{}, fmt.Errorf("extract error: %w", err)
// 	}

// 	// build enriched record
// 	enr := types.EnrichedRecord{
// 		CallRecord:    r,
// 		Extraction:    ext,
// 		VintageBucket: vintageBucket(r.VintageMonth),
// 	}
// 	return enr, nil
// }

// func vintageBucket(v int) string {
// 	switch {
// 	case v <= 2:
// 		return "0-2M"
// 	case v <= 6:
// 		return "2-6M"
// 	case v <= 12:
// 		return "6-12M"
// 	default:
// 		return "12+M"
// 	}
// }
