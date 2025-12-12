package aggregator

import "voice-insights-go/internal/types"

type Insight struct {
	ConfusionByVintage map[string]float64 `json:"confusion_by_vintage"`
	CategoryCounts     map[string]int     `json:"category_counts"`
}

func Aggregate(records []types.EnrichedRecord) Insight {
	total := map[string]int{}
	confused := map[string]int{}
	cats := map[string]int{}
	for _, r := range records {
		vb := r.VintageBucket
		total[vb]++
		if r.IsConfused {
			confused[vb]++
		}
		if r.Category != "" {
			cats[r.Category]++
		}
	}
	confRate := map[string]float64{}
	for k := range total {
		if total[k] > 0 {
			confRate[k] = float64(confused[k]) / float64(total[k])
		} else {
			confRate[k] = 0
		}
	}
	return Insight{ConfusionByVintage: confRate, CategoryCounts: cats}
}
