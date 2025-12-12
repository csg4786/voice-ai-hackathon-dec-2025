package dataset

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"
	"voice-insights-go/internal/logger"
)

type DatasetSummary struct {
	TotalCalls            int                 `json:"total_calls"`
	ByCategory            map[string]int      `json:"by_category"`
	ByCityTopN            map[string][]string `json:"by_city_top_issues"`
	ByVintageBucket       map[string]float64  `json:"by_vintage_bucket_rate"`
	TopExampleTranscripts []string            `json:"top_example_transcripts"`
}

// LoadAndSummarize reads the dataset and produces a compact summary used as LLM context.
func LoadAndSummarize(path string) (DatasetSummary, error) {
	log := logger.New().WithField("component", "dataset.summary").WithField("path", path)
	log.Info("opening dataset for summarization")
	f, err := excelize.OpenFile(path)
	if err != nil {
		log.WithError(err).Error("open failed")
		return DatasetSummary{}, fmt.Errorf("open: %w", err)
	}
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		log.Error("no sheets found")
		return DatasetSummary{}, fmt.Errorf("no sheets")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		log.WithError(err).Error("read rows failed")
		return DatasetSummary{}, fmt.Errorf("read: %w", err)
	}
	if len(rows) <= 1 {
		log.Error("no data rows")
		return DatasetSummary{}, fmt.Errorf("no data rows")
	}

	complaintTokens := []string{"price", "refund", "payment", "delivery", "broken", "verification", "register", "gst", "tax", "quality"}
	byCat := map[string]int{}
	byCityCounts := map[string]map[string]int{}
	byVintageCount := map[string]int{}
	byVintageConf := map[string]int{}
	examples := []string{}

	header := rows[0]
	transcriptIdx := -1
	cityIdx := -1
	vintageIdx := -1
	for i, h := range header {
		n := strings.ToLower(strings.TrimSpace(h))
		if transcriptIdx == -1 && (strings.Contains(n, "transcript") || strings.Contains(n, "text")) {
			transcriptIdx = i
		}
		if cityIdx == -1 && strings.Contains(n, "city") {
			cityIdx = i
		}
		if vintageIdx == -1 && (strings.Contains(n, "vintage") || strings.Contains(n, "month")) {
			vintageIdx = i
		}
	}
	if transcriptIdx == -1 {
		if len(header) > 5 {
			transcriptIdx = 5
		} else {
			transcriptIdx = -1
		}
	}
	log.WithFields(map[string]interface{}{
		"transcriptIdx": transcriptIdx,
		"cityIdx":       cityIdx,
		"vintageIdx":    vintageIdx,
	}).Info("detected summary column indices")

	for i, r := range rows {
		if i == 0 {
			continue
		}
		text := ""
		city := ""
		vintage := ""
		if transcriptIdx >= 0 && transcriptIdx < len(r) {
			text = r[transcriptIdx]
		}
		if cityIdx >= 0 && cityIdx < len(r) {
			city = strings.ToLower(strings.TrimSpace(r[cityIdx]))
		}
		if vintageIdx >= 0 && vintageIdx < len(r) {
			vintage = strings.TrimSpace(r[vintageIdx])
		}
		lower := strings.ToLower(text)
		cat := "other"
		for _, t := range complaintTokens {
			if strings.Contains(lower, t) {
				cat = t
				break
			}
		}
		byCat[cat]++
		if city != "" {
			if _, ok := byCityCounts[city]; !ok {
				byCityCounts[city] = map[string]int{}
			}
			for _, t := range complaintTokens {
				if strings.Contains(lower, t) {
					byCityCounts[city][t]++
				}
			}
		}
		// vintage bucket
		bucket := "12+"
		if vintage != "" {
			var vi int
			fmt.Sscanf(vintage, "%d", &vi)
			switch {
			case vi <= 2:
				bucket = "0-2"
			case vi <= 6:
				bucket = "2-6"
			case vi <= 12:
				bucket = "6-12"
			default:
				bucket = "12+"
			}
		}
		byVintageCount[bucket]++
		if strings.Contains(lower, "confus") || strings.Contains(lower, "don't understand") || strings.Contains(lower, "how to") {
			byVintageConf[bucket]++
		}
		if len(examples) < 6 && text != "" {
			examples = append(examples, text)
		}
	}
	byCityTopN := map[string][]string{}
	for city, m := range byCityCounts {
		type pc struct{ p string; c int }
		var arr []pc
		for k, v := range m {
			arr = append(arr, pc{k, v})
		}
		sort.Slice(arr, func(i, j int) bool { return arr[i].c > arr[j].c })
		top := []string{}
		for i := 0; i < len(arr) && i < 3; i++ {
			top = append(top, arr[i].p)
		}
		byCityTopN[city] = top
	}
	byVintageRate := map[string]float64{}
	for k, tot := range byVintageCount {
		if tot == 0 {
			byVintageRate[k] = 0
		} else {
			byVintageRate[k] = float64(byVintageConf[k]) / float64(tot)
		}
	}

	ds := DatasetSummary{
		TotalCalls:            len(rows) - 1,
		ByCategory:            byCat,
		ByCityTopN:            byCityTopN,
		ByVintageBucket:       byVintageRate,
		TopExampleTranscripts: examples,
	}
	log.WithFields(map[string]interface{}{
		"total_calls": ds.TotalCalls,
		"categories":  len(ds.ByCategory),
		"cities":      len(ds.ByCityTopN),
	}).Info("dataset summarization complete")
	// log top example transcripts (full)
	for i, ex := range ds.TopExampleTranscripts {
		log.WithField("example_index", i).Debug("example transcript", ex)
	}
	return ds, nil
}
