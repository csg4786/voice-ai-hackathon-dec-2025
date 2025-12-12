// internal/types/kpi_models.go
package types

// --------------------------------------------
// Top-level request for the audio analysis API
// --------------------------------------------
type TranscriptRequest struct {
    AudioURL   string `json:"audio_url"`
    Transcript string `json:"transcript"`
}

// --------------------------------------------
// FINAL output delivered to frontend
// --------------------------------------------
type TranscriptResponse struct {
    AudioURL      string         `json:"audio_url"`
    Transcript    string         `json:"transcript"`
    KPIExtraction KPIExtraction  `json:"kpi_extraction"`
    Evidence      EvidenceSummary `json:"evidence"`
    DurationMs    int64           `json:"duration_ms"`
}

// ============================================================
//  IMPORTANT: This wrapper is required because
//  extractor/advanced.go uses KPIExtraction as a top-level type.
// ============================================================

type KPIExtraction struct {
    CustomerProblem CustomerProblem `json:"customer_problem"`
    AgentAnalysis   AgentAnalysis   `json:"agent_analysis"`
    KPI             KPIBlock        `json:"kpi"`
    ShouldHaveDone  ShouldHaveDone  `json:"should_have_done"`
    DatasetInsights DatasetInsights `json:"dataset_insights"`
}

// --------------------------------------------
// Customer problem block
// --------------------------------------------
type CustomerProblem struct {
    PrimaryIssue     string `json:"primary_issue"`
    IssueDescription string `json:"issue_description"`
    UrgencyLevel     string `json:"urgency_level"`
    Severity         int    `json:"severity"` // 1–5
}

// --------------------------------------------
// Agent analysis block
// --------------------------------------------
type AgentAnalysis struct {
    StepsExplainedByAgent []string `json:"steps_explained_by_agent"`
    CorrectnessOfGuidance bool     `json:"correctness_of_guidance"`
    MissedOpportunities   []string `json:"missed_opportunities"`
    AgentSentiment        string   `json:"agent_sentiment"`
    ComplianceFlags       []string `json:"compliance_flags"`
}

// --------------------------------------------
// KPI block
// --------------------------------------------
type KPIBlock struct {
    CustomerTalkRatio float64 `json:"customer_talk_ratio"`
    AgentTalkRatio    float64 `json:"agent_talk_ratio"`
    SilenceSeconds    int     `json:"silence_seconds"`
    InterruptionCount int     `json:"interruption_count"`
    FrustrationScore  float64 `json:"frustration_score"` // 0–1
    ConfusionLevel    float64 `json:"confusion_level"`   // 0–1
}

// --------------------------------------------
// Recommended actions
// --------------------------------------------
type ShouldHaveDone struct {
    IdealResolutionPath string `json:"ideal_resolution_path"`
    RecommendedFollowUp string `json:"recommended_followup"`
    DepartmentOwner     string `json:"department_owner"`
}

// --------------------------------------------
// Dataset-driven insights
// --------------------------------------------
type DatasetInsights struct {
    SimilarCallsCount int    `json:"similar_calls_count"`
    CityTrend         string `json:"city_trend"`
    VintageTrend      string `json:"vintage_trend"`
    ProbableRootCause string `json:"probable_root_cause"`
}

// --------------------------------------------
// Evidence summary
// --------------------------------------------
type EvidenceSummary struct {
    CityTopIssues         []string `json:"city_top_issues"`
    DatasetTotalCalls     int      `json:"dataset_total_calls"`
    MatchedCity           string   `json:"matched_city"`
    ProbableVintageBucket string   `json:"probable_vintage_bucket"`
    SimilarCallsCountEst  int      `json:"similar_calls_count_estimate"`
}

// --------------------------------------------
// Dataset summary (pre-processed at startup)
// --------------------------------------------
type DatasetSummary struct {
    TotalCalls     int      `json:"total_calls"`
    CityTopIssues  []string `json:"city_top_issues"`
    VintageBuckets []string `json:"vintage_buckets"`
}