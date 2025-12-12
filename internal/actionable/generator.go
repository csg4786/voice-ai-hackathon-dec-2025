package actionable

import (
	"fmt"
	"voice-insights-go/internal/aggregator"
)

type ActionCard struct {
	Insight string `json:"insight"`
	Action  string `json:"action"`
	Impact  string `json:"impact"`
}

func Generate(ins aggregator.Insight) ActionCard {
	worst := ""
	highest := 0.0
	for b, v := range ins.ConfusionByVintage {
		if v > highest {
			highest = v
			worst = b
		}
	}
	if highest >= 0.35 && worst != "" {
		return ActionCard{
			Insight: fmt.Sprintf("High confusion in %s (%.0f%%)", worst, highest*100),
			Action:  "Deploy onboarding voice guide for new sellers; set proactive verification",
			Impact:  "Reduce repeat escalations and support load",
		}
	}
	return ActionCard{
		Insight: "No strong confusion pattern detected",
		Action:  "Monitor and collect more data",
		Impact:  "Low immediate intervention",
	}
}
