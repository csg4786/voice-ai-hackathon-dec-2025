# kpi_tooltips.py
# Centralized Tooltip Dictionary for all KPIs + insights

TOOLTIPS = {
    # -------------------- KPI METRICS --------------------
    "customer_talk_ratio": "Portion of the conversation spoken by the customer.",
    "agent_talk_ratio": "Portion of conversation spoken by the agent.",
    "frustration_score": "Emotionally inferred frustration level of the customer.",
    "confusion_level": "Likelihood that the customer is confused or unclear.",
    "empathy_score": "How empathetic the agent sounded.",
    "resolution_likelihood": "Probability that the issue will be resolved successfully.",
    "avg_sentence_length_customer": "Average length of customer sentences. Longer may indicate venting.",
    "avg_sentence_length_agent": "Average agent sentence length. Shorter often means clarity.",
    "silence_seconds": "Total duration of silence in the call.",
    "interruption_count": "How often speakers talked over each other.",
    "dead_air_instances": "Silent gaps where no one was speaking.",
    "topic_switch_count": "Number of times the conversation changed topics abruptly.",

    # -------------------- CUSTOMER PROBLEM --------------------
    "primary_issue": "The main concern expressed by the customer.",
    "issue_description": "Detailed explanation of the customer's complaint.",
    "urgency_level": "How urgent the problem is for the customer.",
    "severity": "Severity level of the complaint.",
    "customer_intent": "What the customer ultimately wants to achieve.",
    "repeat_issue": "Indicates whether the customer has reported this problem before.",
    "related_issue_category": "High-level issue category such as Lead Quality or Billing.",

    # -------------------- AGENT ANALYSIS --------------------
    "steps_explained_by_agent": "Steps or explanations provided by the agent.",
    "correctness_of_guidance": "Whether the agent's explanation was correct.",
    "missed_opportunities": "Important moments where the agent should have acted but didn't.",
    "agent_sentiment": "Emotional tone of the agent.",
    "rapport_score": "How well the agent built rapport.",
    "professionalism_score": "Agent's politeness and professionalism.",
    "solution_accuracy_score": "Accuracy of solutions suggested by the agent.",
    "agent_confidence_level": "Perceived confidence of the agent.",

    # -------------------- SHOULD HAVE DONE --------------------
    "ideal_resolution_path": "What the ideal conversation flow should have been.",
    "recommended_followup": "What should happen next after the call.",
    "department_owner": "Which team should take responsibility.",
    "crucial_missed_questions": "Important questions the agent failed to ask.",
    "required_data_points_not_collected": "Important data points the agent didn't collect.",

    # -------------------- ACTIONS --------------------
    "executive_actions_required": "Actions required by sales executives.",
    "customer_actions_required": "Actions the customer must take.",
    "system_actions_required": "Actions the backend system must take.",
    "priority": "Priority level of the issue.",
    "requires_escalation": "Whether this conversation must be escalated.",
    "escalation_reason": "Why escalation is needed.",

    # -------------------- CONVERSATION QUALITY --------------------
    "overall_score": "Overall quality score for the conversation.",
    "clarity_score": "How clear the agent's communication was.",
    "listening_score": "How well the agent listened.",
    "relevance_score": "Relevance of responses to customer issues.",
    "trust_building_score": "How much trust the agent built.",
    "red_flags": "Critical negative signals detected in the call.",

    # -------------------- TRENDS --------------------
    "similar_calls_count": "Number of similar historical calls found.",
    "dominant_issue_category": "Most common issue category in similar calls.",
    "city_trend": "Geographical trend among similar calls.",
    "vintage_trend": "Customer account age trend.",
    "engagement_level_trend": "Engagement levels across similar cases.",
    "actionability_pattern": "Patterns in how similar issues were resolved.",
    "probable_root_cause": "Likely root cause based on trends.",
    "recommended_playbook": "Recommended steps for resolution.",
    "historical_resolution_rate": "Success rate for resolving similar calls.",
    "historical_escalation_rate": "Escalation frequency for similar calls.",

    # -------------------- BUSINESS IMPACT --------------------
    "risk_of_churn": "Likelihood that this customer will leave.",
    "revenue_opportunity_loss": "Revenue lost due to dissatisfaction.",
    "customer_ltv_bucket": "Customer lifetime value segment.",
    "service_gap_identified": "Gap in service quality or delivery.",
    "fix_urgency_level": "Urgency with which the issue must be fixed.",
}
