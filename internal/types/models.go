package types

// central types used across the pipeline and returned to clients
type CustomerProblem struct {
	PrimaryIssue    string `json:"primary_issue"`
	IssueDescription string `json:"issue_description"`
	UrgencyLevel     string `json:"urgency_level"`
	Severity         int    `json:"severity"`
}

type AgentAnalysis struct {
	StepsExplainedByAgent []string `json:"steps_explained_by_agent"`
	CorrectnessOfGuidance bool     `json:"correctness_of_guidance"`
	MissedOpportunities   []string `json:"missed_opportunities"`
	AgentSentiment        string   `json:"agent_sentiment"`
	ComplianceFlags       []string `json:"compliance_flags"`
}

type KPIFields struct {
	CustomerTalkRatio float64 `json:"customer_talk_ratio"`
	AgentTalkRatio    float64 `json:"agent_talk_ratio"`
	SilenceSeconds    int     `json:"silence_seconds"`
	InterruptionCount int     `json:"interruption_count"`
	FrustrationScore  float64 `json:"frustration_score"`
	ConfusionLevel    float64 `json:"confusion_level"`
}

type ShouldHaveDone struct {
	IdealResolutionPath string `json:"ideal_resolution_path"`
	RecommendedFollowUp string `json:"recommended_followup"`
	DepartmentOwner     string `json:"department_owner"`
}

type DatasetInsights struct {
	SimilarCallsCount int    `json:"similar_calls_count"`
	CityTrend         string `json:"city_trend"`
	VintageTrend      string `json:"vintage_trend"`
	ProbableRootCause string `json:"probable_root_cause"`
}

// Top-level structure returned by the LLM extraction step
type KPIExtraction struct {
	CustomerProblem CustomerProblem  `json:"customer_problem"`
	AgentAnalysis   AgentAnalysis    `json:"agent_analysis"`
	KPI             KPIFields        `json:"kpi"`
	ShouldHaveDone  ShouldHaveDone   `json:"should_have_done"`
	DatasetInsights DatasetInsights  `json:"dataset_insights"`
}

// processor response type (what /process returns)
type KPIResult struct {
	AudioURL   string                 `json:"audio_url"`
	Transcript string                 `json:"transcript"`
	KPI        KPIExtraction          `json:"kpi_extraction"`
	Evidence   map[string]interface{} `json:"evidence"`
	DurationMs int64                  `json:"duration_ms"`
	Error      string                 `json:"error,omitempty"`
}

// original dataset row (basic)
type CallRecord struct {
	CallID       string `json:"call_id"`
	CallType     string `json:"call_type"`
	AudioURL     string `json:"audio_url"`
	City         string `json:"city"`
	VintageMonth int    `json:"vintage_month"`
	RepeatEsc    int    `json:"repeat_esc"`
}

// optional enriched record (for aggregator if used)
type EnrichedRecord struct {
	CallID        string `json:"call_id"`
	VintageBucket string `json:"vintage_bucket"`
	IsConfused    bool   `json:"is_confused"`
	Category      string `json:"category"`
}
