package types

type CallRecord struct {
	CallID       string `json:"call_id"`
	CallType     string `json:"call_type,omitempty"`
	AudioURL     string `json:"audio_url"`
	City         string `json:"city,omitempty"`
	VintageMonth int    `json:"vintage_month,omitempty"`
	RepeatEsc    int    `json:"repeat_escalation,omitempty"`
	Transcript   string `json:"transcript,omitempty"`
}

type Extraction struct {
	Category         string `json:"category"`
	IsConfused       bool   `json:"is_confused"`
	Sentiment        string `json:"sentiment"`
	EscalationReason string `json:"escalation_reason"`
	RootCause        string `json:"root_cause"`
	NextBestAction   string `json:"next_best_action"`
}

type EnrichedRecord struct {
	CallRecord
	Extraction
	VintageBucket string `json:"vintage_bucket,omitempty"`
}
