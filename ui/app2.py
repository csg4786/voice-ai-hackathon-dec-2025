import streamlit as st
import requests
import matplotlib.pyplot as plt
import numpy as np
import json
import re

from textblob import TextBlob
from speaker_classifier import infer_roles

# ------------------------------- CONFIG -------------------------------
BACKEND_URL = "http://localhost:8080"
st.set_page_config(page_title="Voice Insights AI", layout="wide")

# ------------------------------- TITLE -------------------------------
st.title("🎙️ Voice Insights AI - Complete Call Intelligence Dashboard")
st.caption("Transcription • Diarization • KPIs • Sentiment • Actionable Insights")

# -------------------------------------------------------------------
# UTILITY: DIARIZATION
# -------------------------------------------------------------------

def diarize(transcript: str):
    """
    Returns structured diarization format:
    [
      { speaker_id, role, text, sentiment, timestamp }
    ]
    """

    role_map = infer_roles(transcript)
    lines = transcript.split("\n")

    diarized = []
    ts = 0.0  # synthetic timeline

    for line in lines:
        line = line.strip()
        if not line:
            continue

        match = re.match(r"(Speaker\s+\d+)\s*:\s*(.*)", line, re.I)
        if not match:
            continue

        speaker_id = match.group(1)
        text = match.group(2).strip()

        role = role_map.get(speaker_id, "Unknown")
        sentiment = float(TextBlob(text).sentiment.polarity)

        diarized.append({
            "speaker_id": speaker_id,
            "role": role,
            "text": text,
            "timestamp": ts,
            "sentiment": sentiment
        })

        ts += np.random.uniform(3.0, 6.0)  # smooth synthetic timeline spacing

    return diarized


# -------------------------------------------------------------------
# FRONTEND INPUT SECTION
# -------------------------------------------------------------------

st.subheader("🔗 Enter Call Recording URL")
audio_url = st.text_input("Call Recording URL", placeholder="https://example.com/audio.mp3")

if st.button("Analyze Call"):
    if not audio_url.strip():
        st.error("Please enter a valid audio URL.")
        st.stop()

    # ------------------------------- AUDIO PLAYER -------------------------------
    st.subheader("🎧 Audio Playback")
    st.audio(audio_url)

    with st.spinner("Transcribing and analyzing..."):
        try:
            resp = requests.get(
                f"{BACKEND_URL}/process",
                params={"audio_url": audio_url},
                timeout=150
            )
            data = resp.json()
        except Exception as e:
            st.error(f"Backend unreachable: {e}")
            st.stop()

    # ERROR HANDLING
    if "error" in data and data["error"]:
        st.error("Processing failed: " + data["error"])
        st.json(data)
        st.stop()

    # Extract response contents
    transcript = data.get("transcript", "")
    kpi = data.get("kpi", {})
    evidence = data.get("evidence", {})

    if not transcript:
        st.warning("No transcript returned from backend.")
        st.stop()

    # -------------------------------------------------------------------
    # DIARIZE TRANSCRIPT
    # -------------------------------------------------------------------
    diarized = diarize(transcript)

    # ------------------------------- TRANSCRIPT (CHAT STYLE) -------------------------------
    st.subheader("💬 Speaker Transcript")

    with st.expander("Show Transcript"):
        for turn in diarized:
            speaker = turn["role"]
            text_line = turn["text"]

            color = "#38C744" if speaker == "Agent" else "#3A8DC9"
            label = "Agent" if speaker == "Agent" else "Customer"

            st.markdown(
                f"""
                <div style="background:{color};padding:10px;border-radius:10px;margin-bottom:6px;">
                    <b>{label}:</b> {text_line}
                </div>
                """,
                unsafe_allow_html=True,
            )

    st.markdown("---")

    # ------------------------------- CONVERSATION TIMELINE -------------------------------
    st.subheader("🕒 Conversation Timeline")

    timestamps = [t["timestamp"] for t in diarized]
    roles = [t["role"] for t in diarized]
    timeline_points = [1 if r == "Agent" else 0 for r in roles]

    fig, ax = plt.subplots(figsize=(10, 2))
    ax.plot(timestamps, timeline_points, marker="o", linestyle="-")
    ax.set_yticks([0, 1])
    ax.set_yticklabels(["Customer", "Agent"])
    ax.set_xlabel("Time (synthetic)")
    ax.set_title("Speaker Participation Timeline")
    st.pyplot(fig)

    st.markdown("---")

    # ------------------------------- FRUSTRATION TRAJECTORY -------------------------------
    st.subheader("🔥 Customer Frustration Trajectory")

    sentiments = [t["sentiment"] for t in diarized]
    if len(sentiments) > 1:
        sentiments_norm = (np.array(sentiments) - np.min(sentiments)) / (
            np.ptp(sentiments) + 1e-9
        )
    else:
        sentiments_norm = [0.5]

    fig2, ax2 = plt.subplots(figsize=(10, 3))
    ax2.plot(sentiments_norm, color="red")
    ax2.set_ylim(0, 1)
    ax2.set_title("Sentiment Curve (Lower = More Frustrated)")
    st.pyplot(fig2)

    st.markdown("---")

    st.subheader("Raw Backend Response")
    st.json(data)


    # ------------------------------- KPI SECTION -------------------------------
    st.subheader("📊 Key Metrics")

    # detect structure: nested under "kpi" OR flat at root
    kpi_root = data.get("kpi", data)

    cust = kpi_root.get("customer_problem", {
        "primary_issue": "N/A",
        "severity": 0,
        "urgency_level": "N/A",
    })

    agent = kpi_root.get("agent_analysis", {
        "steps_explained_by_agent": [],
        "missed_opportunities": [],
        "compliance_flags": []
    })

    k = kpi_root.get("kpi", {
        "customer_talk_ratio": 0,
        "agent_talk_ratio": 0,
        "silence_seconds": 0,
        "frustration_score": 0,
        "confusion_level": 0,
        "interruption_count": 0,
    })

    sh = kpi_root.get("should_have_done", {
        "ideal_resolution_path": "N/A",
        "recommended_followup": "N/A",
        "department_owner": "N/A"
    })

    ds_ins = kpi_root.get("dataset_insights", {})

    # ------------------------------- WHAT SHOULD HAVE BEEN DONE -------------------------------
    st.subheader("🛠 Recommended Resolution Path")
    st.markdown(f"**Ideal Path:** {sh['ideal_resolution_path']}")
    st.markdown(f"**Recommended Follow-up:** {sh['recommended_followup']}")
    st.markdown(f"**Department Owner:** {sh['department_owner']}")

    st.markdown("---")

    # ------------------------------- AGENT ANALYSIS -------------------------------
    st.subheader("🎧 Agent Analysis")

    st.markdown("**Steps Explained by Agent:**")
    for step in agent["steps_explained_by_agent"]:
        st.write(f"- {step}")

    st.markdown("**Missed Opportunities:**")
    for mo in agent["missed_opportunities"]:
        st.write(f"- {mo}")

    st.markdown("**Compliance Flags:**")
    for cf in agent["compliance_flags"]:
        st.write(f"- {cf}")

    st.markdown("---")

    # ------------------------------- DATASET INSIGHTS -------------------------------
    st.subheader("📈 Dataset Insights Used in Reasoning")
    st.json(ds_ins)

    st.subheader("📁 Evidence Extracted")
    st.json(evidence)

    st.success("✔ Completed Analysis")


