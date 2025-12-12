# app2.py
import streamlit as st
import requests
import matplotlib.pyplot as plt
import numpy as np
import json
import re
from textblob import TextBlob

# ------------------------------- CONFIG -------------------------------
BACKEND_URL = "http://localhost:8080"
st.set_page_config(page_title="Voice Insights AI", layout="wide")

# ------------------------------- TITLE -------------------------------
st.title("🎙️ Voice Insights AI - Complete Call Intelligence Dashboard")
st.caption("Transcription • Diarization (Speaker 1 / Speaker 2) • KPIs • Sentiment • Actionable Insights")

# ------------------------------- DIARIZATION -------------------------------
def diarize_keep_transcript_speakers(transcript: str):
    """
    Parse transcript into a list of turns:
    [
      { "speaker": "Speaker 1", "text": "...", "timestamp": float, "sentiment": float }
    ]
    Keeps speaker labels exactly as they appear in the transcript.
    Produces synthetic timestamps in order of appearance.
    """

    lines = transcript.splitlines()
    diarized = []
    ts = 0.0

    speaker_pattern = re.compile(r'^(Speaker\s*\d+)\s*[:\-]\s*(.*)$', flags=re.I)

    for raw in lines:
        line = raw.strip()
        if not line:
            continue
        m = speaker_pattern.match(line)
        if not m:
            # If line doesn't match speaker pattern, consider it as continuaton of last speaker
            if diarized:
                diarized[-1]["text"] += " " + line
                # recompute sentiment
                try:
                    diarized[-1]["sentiment"] = float(TextBlob(diarized[-1]["text"]).sentiment.polarity)
                except Exception:
                    diarized[-1]["sentiment"] = 0.0
            continue

        speaker = m.group(1)
        text = m.group(2).strip()

        try:
            sentiment = float(TextBlob(text).sentiment.polarity)
        except Exception:
            sentiment = 0.0

        diarized.append({
            "speaker": speaker,
            "text": text,
            "timestamp": ts,
            "sentiment": sentiment
        })

        # increment synthetic time (approx per turn)
        ts += float(np.random.uniform(3.0, 6.0))

    return diarized

# ------------------------------- KPI PARSING -------------------------------
def safe_get(d: dict, path: list, default=None):
    """ Safe nested get """
    cur = d
    for p in path:
        if not isinstance(cur, dict):
            return default
        cur = cur.get(p, default)
    return cur

def parse_kpi_extraction(kpi_root: dict):
    """
    Return a normalized structure with all expected fields (even if missing).
    """
    cust = kpi_root.get("customer_problem", {}) or {}
    agent = kpi_root.get("agent_analysis", {}) or {}
    kpi = kpi_root.get("kpi", {}) or {}
    sh = kpi_root.get("should_have_done", {}) or {}
    ds = kpi_root.get("dataset_insights", {}) or {}

    parsed = {
        "customer_problem": {
            "primary_issue": cust.get("primary_issue", ""),
            "issue_description": cust.get("issue_description", ""),
            "urgency_level": cust.get("urgency_level", ""),
            "severity": cust.get("severity", 0),
        },
        "agent_analysis": {
            "steps_explained_by_agent": agent.get("steps_explained_by_agent", []),
            "correctness_of_guidance": agent.get("correctness_of_guidance", False),
            "missed_opportunities": agent.get("missed_opportunities", []),
            "agent_sentiment": agent.get("agent_sentiment", "") or agent.get("agent_sentiment", ""),
            "compliance_flags": agent.get("compliance_flags", []),
        },
        "kpi": {
            "customer_talk_ratio": kpi.get("customer_talk_ratio", 0),
            "agent_talk_ratio": kpi.get("agent_talk_ratio", 0),
            "silence_seconds": kpi.get("silence_seconds", 0),
            "interruption_count": kpi.get("interruption_count", 0),
            "frustration_score": kpi.get("frustration_score", 0),
            "confusion_level": kpi.get("confusion_level", 0),
        },
        "should_have_done": {
            "ideal_resolution_path": sh.get("ideal_resolution_path", sh.get("agent_actions", "")) if isinstance(sh, dict) else "",
            "recommended_followup": sh.get("recommended_followup", ""),
            "department_owner": sh.get("department_owner", ""),
            # Some LLM payloads use different keys - handle that:
            "agent_actions": sh.get("agent_actions", []) if isinstance(sh, dict) else []
        },
        "dataset_insights": {
            "similar_calls_count": ds.get("similar_calls_count", ds.get("similar_calls_count", 0)),
            "city_trend": ds.get("city_trend", ""),
            "vintage_trend": ds.get("vintage_trend", ""),
            "probable_root_cause": ds.get("probable_root_cause", ""),
            # additional helpful fields if present
            "overall_call_volume": ds.get("overall_call_volume", None),
            "top_issues_by_category": ds.get("top_issues_by_category", None),
        }
    }
    return parsed

# ------------------------------- FRONTEND INPUT -------------------------------
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

    if "error" in data and data["error"]:
        st.error("Processing failed: " + data["error"])
        st.json(data)
        st.stop()

    transcript = data.get("transcript", "")
    # backend uses "kpi_extraction" top-level key in the examples you provided
    kpi_root_raw = data.get("kpi_extraction", {}) or {}
    evidence = data.get("evidence", {}) or {}

    if not transcript:
        st.warning("No transcript returned from backend.")
        st.stop()

    # -------------------------------------------------------------------
    # DIARIZE (KEEP SPEAKERS AS 'Speaker 1' / 'Speaker 2' EXACTLY)
    # -------------------------------------------------------------------
    diarized = diarize_keep_transcript_speakers(transcript)

    # ------------------------------- TRANSCRIPT DISPLAY -------------------------------
    st.subheader("💬 Speaker Transcript (keeps Speaker IDs exactly as in transcript)")

    with st.expander("Show Transcript (grouped by speaker turns)"):
        for turn in diarized:
            sp = turn["speaker"]
            text_line = turn["text"]
            # Speaker-specific color - keep simple
            color = "#F2F2F2"
            st.markdown(
                f"""
                <div style="background:{color};padding:10px;border-radius:8px;margin-bottom:6px;">
                    <b>{sp}:</b> {text_line}
                </div>
                """,
                unsafe_allow_html=True,
            )

    st.markdown("---")

    # ------------------------------- TIMELINE -------------------------------
    st.subheader("🕒 Conversation Timeline (speaker order)")

    timestamps = [t["timestamp"] for t in diarized]
    speakers = [t["speaker"] for t in diarized]
    # map speaker names to numeric value for simple timeline ordering
    unique_speakers = sorted(list(dict.fromkeys(speakers)))  # keeps first-seen order but sorted for stability
    speaker_to_num = {s: i for i, s in enumerate(unique_speakers)}

    timeline_vals = [speaker_to_num.get(s, 0) for s in speakers]

    fig, ax = plt.subplots(figsize=(10, 2))
    if timestamps and timeline_vals:
        ax.plot(timestamps, timeline_vals, marker="o", linestyle="-")
        ax.set_yticks(list(speaker_to_num.values()))
        ax.set_yticklabels(list(speaker_to_num.keys()))
        ax.set_xlabel("Time (synthetic)")
        ax.set_title("Speaker Participation Timeline")
    else:
        ax.text(0.5, 0.5, "No turns to plot", ha="center")
    st.pyplot(fig)

    st.markdown("---")

    # ------------------------------- SENTIMENT / FRUSTRATION TRAJECTORIES -------------------------------
    st.subheader("🔥 Speaker-wise Sentiment / Frustration Trajectories")

    # Build per-speaker lists preserving chronological order
    speaker_series = {}
    speaker_ts = {}
    for t in diarized:
        s = t["speaker"]
        speaker_series.setdefault(s, []).append(t["sentiment"])
        speaker_ts.setdefault(s, []).append(t["timestamp"])

    # Create side-by-side plots
    cols = st.columns(2)
    speakers_list = list(speaker_series.keys())[:2]  # at most two speakers; if more, show first two
    # Ensure we always show Speaker 1 and Speaker 2 if present
    if "Speaker 1" in speaker_series and "Speaker 2" in speaker_series:
        speakers_list = ["Speaker 1", "Speaker 2"]
    elif len(speaker_series) == 1:
        speakers_list = [list(speaker_series.keys())[0], None]
    else:
        # pad to two
        while len(speakers_list) < 2:
            speakers_list.append(None)

    for idx, sp in enumerate(speakers_list):
        with cols[idx]:
            if sp is None:
                st.write("")  # empty placeholder
                continue
            s_vals = np.array(speaker_series.get(sp, []), dtype=float)
            s_ts = speaker_ts.get(sp, list(range(len(s_vals))))
            st.markdown(f"**{sp} Sentiment (polarity)**")
            if s_vals.size == 0:
                st.info("No turns for this speaker.")
                continue

            # normalize to 0..1 for plotting (polarity is -1..1)
            minv = np.min(s_vals)
            maxv = np.max(s_vals)
            if np.isclose(maxv, minv):
                norm = np.full_like(s_vals, 0.5)
            else:
                norm = (s_vals - minv) / (maxv - minv)

            fig_s, ax_s = plt.subplots(figsize=(6, 3))
            ax_s.plot(s_ts, norm, marker="o")
            ax_s.set_ylim(0, 1)
            ax_s.set_xlabel("Time (synthetic)")
            ax_s.set_title(f"{sp} sentiment (normalized - 0 to 1)")
            st.pyplot(fig_s)

            # small metrics
            avg_sent = float(np.mean(s_vals)) if s_vals.size else 0.0
            st.metric(label="Avg polarity", value=f"{avg_sent:.2f}")
            st.metric(label="Turns", value=len(s_vals))

    st.markdown("---")

    # ------------------------------- RAW BACKEND RESPONSE (debug) -------------------------------
    st.subheader("Raw Backend Response (for debugging)")
    with st.expander("Show raw JSON response"):
        st.json(data)

    # ------------------------------- KPI DATA ACCESS -------------------------------
    parsed_kpi = parse_kpi_extraction(kpi_root_raw)

    # ------------------------------- GROUPED KPI UI -------------------------------
    st.subheader("📊 KPI & Insights (grouped)")

    # Customer Problem card
    cp = parsed_kpi["customer_problem"]
    st.markdown("### 🔎 Customer Problem")
    st.markdown(f"**Primary Issue**  \n{cp['primary_issue'] or 'N/A'}")
    st.markdown(f"**Urgency**  \n{cp['urgency_level'] or 'N/A'}")
    st.markdown(f"**Severity**  \n{cp['severity']}")
    st.markdown("**Issue Description**")
    st.write(cp["issue_description"] or "N/A")

    st.markdown("---")

    # Agent Analysis
    aa = parsed_kpi["agent_analysis"]
    st.markdown("### 🎧 Agent Analysis")
    st.markdown("**Agent Sentiment (LLM)**")
    st.write(aa.get("agent_sentiment", "") or "N/A")

    st.markdown("**Steps Explained by Agent**")
    if aa["steps_explained_by_agent"]:
        for s in aa["steps_explained_by_agent"]:
            st.write(f"- {s}")
    else:
        st.write("N/A")

    st.markdown("**Missed Opportunities**")
    if aa["missed_opportunities"]:
        for m in aa["missed_opportunities"]:
            st.write(f"- {m}")
    else:
        st.write("N/A")

    st.markdown("**Compliance Flags**")
    if aa["compliance_flags"]:
        for c in aa["compliance_flags"]:
            st.write(f"- {c}")
    else:
        st.write("None")

    st.markdown("---")

    # KPI details (numbers)
    k = parsed_kpi["kpi"]
    st.markdown("### 📈 KPI Details (numeric)")
    # display the small KPIs as metrics in columns
    kcols = st.columns(6)
    kcols[0].metric("Customer talk ratio", f"{k.get('customer_talk_ratio', 0):.2f}")
    kcols[1].metric("Agent talk ratio", f"{k.get('agent_talk_ratio', 0):.2f}")
    kcols[2].metric("Silence (s)", f"{k.get('silence_seconds', 0)}")
    kcols[3].metric("Interruptions", f"{k.get('interruption_count', 0)}")
    kcols[4].metric("Frustration score", f"{k.get('frustration_score', 0):.2f}")
    kcols[5].metric("Confusion level", f"{k.get('confusion_level', 0):.2f}")

    st.markdown("---")

    # Should Have Done
    sh = parsed_kpi["should_have_done"]
    st.markdown("### 🛠 Recommended Resolution Path")
    st.markdown(f"**Ideal Path:**")
    st.write(sh.get("ideal_resolution_path", "N/A"))
    st.markdown(f"**Recommended Follow-up:**")
    st.write(sh.get("recommended_followup", "N/A"))
    st.markdown(f"**Department Owner:** {sh.get('department_owner', 'N/A')}")

    # Agent actions (if LLM provided a list under should_have_done.agent_actions)
    if sh.get("agent_actions"):
        st.markdown("**Agent actions that should have been taken**")
        for a in sh["agent_actions"]:
            st.write(f"- {a}")

    st.markdown("---")

    # Dataset Insights
    ds = parsed_kpi["dataset_insights"]
    st.markdown("### 📈 Dataset Insights")
    st.write("Similar calls (est):", ds.get("similar_calls_count", "N/A"))
    if ds.get("overall_call_volume"):
        st.write("Overall call volume:", ds.get("overall_call_volume"))
    if ds.get("top_issues_by_category"):
        st.markdown("Top issues by category (partial):")
        st.json(ds.get("top_issues_by_category"))

    st.markdown("**City trend:**")
    st.write(ds.get("city_trend", "N/A"))
    st.markdown("**Vintage trend:**")
    st.write(ds.get("vintage_trend", "N/A"))
    st.markdown("**Probable root cause:**")
    st.write(ds.get("probable_root_cause", "N/A"))

    st.markdown("---")

    # Evidence
    st.subheader("📁 Evidence & Grounding")
    st.json(evidence)

    st.success("✔ Completed Analysis")
