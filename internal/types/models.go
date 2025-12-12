package types

// -------------------------
// CUSTOMER PROBLEM
// -------------------------
type CustomerProblem struct {
	PrimaryIssue         string `json:"primary_issue"`
	IssueDescription     string `json:"issue_description"`
	UrgencyLevel         string `json:"urgency_level"`
	Severity             int    `json:"severity"`
	CustomerIntent       string `json:"customer_intent"`
	RepeatIssue          bool   `json:"repeat_issue"`
	RelatedIssueCategory string `json:"related_issue_category"`
}

// -------------------------
// AGENT ANALYSIS
// -------------------------
type AgentAnalysis struct {
	StepsExplainedByAgent []string `json:"steps_explained_by_agent"`
	CorrectnessOfGuidance bool     `json:"correctness_of_guidance"`
	MissedOpportunities   []string `json:"missed_opportunities"`
	AgentSentiment        string   `json:"agent_sentiment"`
	ComplianceFlags       []string `json:"compliance_flags"`

	// Newly added fields for Schema v2
	RapportScore         float64 `json:"rapport_score"`
	ProfessionalismScore float64 `json:"professionalism_score"`
	SolutionAccuracyScore float64 `json:"solution_accuracy_score"`
	AgentConfidenceLevel string  `json:"agent_confidence_level"`
}

// -------------------------
// KPI FIELDS
// -------------------------
type KPIFields struct {
	CustomerTalkRatio        float64 `json:"customer_talk_ratio"`
	AgentTalkRatio           float64 `json:"agent_talk_ratio"`
	SilenceSeconds           int     `json:"silence_seconds"`
	InterruptionCount        int     `json:"interruption_count"`
	FrustrationScore         float64 `json:"frustration_score"`
	ConfusionLevel           float64 `json:"confusion_level"`

	// New Schema v2 fields
	EmpathyScore             float64 `json:"empathy_score"`
	ResolutionLikelihood     float64 `json:"resolution_likelihood"`
	AvgSentenceLengthCustomer float64 `json:"avg_sentence_length_customer"`
	AvgSentenceLengthAgent    float64 `json:"avg_sentence_length_agent"`
	DeadAirInstances          int     `json:"dead_air_instances"`
	TopicSwitchCount          int     `json:"topic_switch_count"`
}

// -------------------------
// SHOULD HAVE DONE
// -------------------------
type ShouldHaveDone struct {
	IdealResolutionPath         string   `json:"ideal_resolution_path"`
	RecommendedFollowUp         string   `json:"recommended_followup"`
	DepartmentOwner             string   `json:"department_owner"`
	CrucialMissedQuestions      []string `json:"crucial_missed_questions"`
	RequiredDataPointsNotCollected []string `json:"required_data_points_not_collected"`
}

// -------------------------
// ACTIONS BLOCK (NEW IN SCHEMA V2)
// -------------------------
type Actions struct {
	ExecutiveActionsRequired []string `json:"executive_actions_required"`
	CustomerActionsRequired  []string `json:"customer_actions_required"`
	SystemActionsRequired    []string `json:"system_actions_required"`

	Priority          string `json:"priority"`
	RequiresEscalation bool   `json:"requires_escalation"`
	EscalationReason  string `json:"escalation_reason"`
}

// -------------------------
// CONVERSATION QUALITY (NEW)
// -------------------------
type ConversationQuality struct {
	OverallScore       float64  `json:"overall_score"`
	ClarityScore       float64  `json:"clarity_score"`
	ListeningScore     float64  `json:"listening_score"`
	RelevanceScore     float64  `json:"relevance_score"`
	TrustBuildingScore float64  `json:"trust_building_score"`
	RedFlags           []string `json:"red_flags"`
}

// -------------------------
// TREND INSIGHTS (SEARCH RESULTS MINING)
// -------------------------
type TrendInsights struct {
	SimilarCallsCount        int     `json:"similar_calls_count"`
	DominantIssueCategory    string  `json:"dominant_issue_category"`
	CityTrend                string  `json:"city_trend"`
	VintageTrend             string  `json:"vintage_trend"`
	EngagementLevelTrend     string  `json:"engagement_level_trend"`
	ActionabilityPattern     string  `json:"actionability_pattern"`
	ProbableRootCause        string  `json:"probable_root_cause"`
	RecommendedPlaybook      string  `json:"recommended_playbook"`
	HistoricalResolutionRate float64 `json:"historical_resolution_rate"`
	HistoricalEscalationRate float64 `json:"historical_escalation_rate"`
}

// -------------------------
// BUSINESS IMPACT (NEW)
// -------------------------
type BusinessImpact struct {
	RiskOfChurn          float64 `json:"risk_of_churn"`
	RevenueOpportunityLoss string `json:"revenue_opportunity_loss"`
	CustomerLTVBucket    string  `json:"customer_ltv_bucket"`
	ServiceGapIdentified string  `json:"service_gap_identified"`
	FixUrgencyLevel      string  `json:"fix_urgency_level"`
}

// -------------------------
// TOP-LEVEL STRUCT (UPDATED)
// -------------------------
type KPIExtraction struct {
	CustomerProblem      CustomerProblem      `json:"customer_problem"`
	AgentAnalysis        AgentAnalysis        `json:"agent_analysis"`
	KPI                  KPIFields            `json:"kpi"`
	ShouldHaveDone       ShouldHaveDone       `json:"should_have_done"`
	Actions              Actions              `json:"actions"`
	ConversationQuality  ConversationQuality  `json:"conversation_quality"`
	TrendInsights        TrendInsights        `json:"trend_insights_from_similar_calls"`
	BusinessImpact       BusinessImpact       `json:"business_impact"`
}

// -------------------------
// EXISTING STRUCTS (UNCHANGED)
// -------------------------
type KPIResult struct {
	AudioURL   string                 `json:"audio_url"`
	Transcript string                 `json:"transcript"`
	KPI        KPIExtraction          `json:"kpi_extraction"`
	Evidence   map[string]interface{} `json:"evidence"`
	DurationMs int64                  `json:"duration_ms"`
	Error      string                 `json:"error,omitempty"`
}

type CallRecord struct {
	CallID       string `json:"call_id"`
	CallType     string `json:"call_type"`
	AudioURL     string `json:"audio_url"`
	City         string `json:"city"`
	VintageMonth int    `json:"vintage_month"`
	RepeatEsc    int    `json:"repeat_esc"`
}

type EnrichedRecord struct {
	CallID        string `json:"call_id"`
	VintageBucket string `json:"vintage_bucket"`
	IsConfused    bool   `json:"is_confused"`
	Category      string `json:"category"`
}
