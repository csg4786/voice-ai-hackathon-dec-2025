def infer_roles(transcript: str):
    """
    Infer which speaker is Customer and which is Agent using hardcoded rules + LLM patterns.

    Rules:
    - Speaker who complains, expresses dissatisfaction → Customer
    - Speaker who explains features, offers help, gives instructions → Agent
    """

    speaker1_score = {"customer": 0, "agent": 0}
    speaker2_score = {"customer": 0, "agent": 0}

    for line in transcript.split("\n"):
        l = line.lower()

        # customer-like patterns
        if any(x in l for x in [
            "not satisfied", "concern", "issue", "problem", "convert nahi",
            "membership", "aap log", "leads", "response nahi", "benefit nahi",
            "maine bola", "nahi ho raha", "dissatisfied", "fee", "bogus"
        ]):
            if l.startswith("speaker 1"):
                speaker1_score["customer"] += 2
            if l.startswith("speaker 2"):
                speaker2_score["customer"] += 2

        # agent-like patterns
        if any(x in l for x in [
            "option batata hu", "aap kijiye", "account manager", "main dekh raha hu",
            "filter lagayiye", "training chahiye", "main guide kar dunga",
            "aap click kijiye", "sir"
        ]):
            if l.startswith("speaker 1"):
                speaker1_score["agent"] += 2
            if l.startswith("speaker 2"):
                speaker2_score["agent"] += 2

    # Decide final roles
    speaker1_role = "Customer" if speaker1_score["customer"] > speaker1_score["agent"] else "Agent"
    speaker2_role = "Customer" if speaker2_score["customer"] > speaker2_score["agent"] else "Agent"

    return {
        "Speaker 1": speaker1_role,
        "Speaker 2": speaker2_role,
    }
