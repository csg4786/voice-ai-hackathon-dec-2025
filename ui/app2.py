# app2.py — FINAL VERSION (Using tooltip dictionary file)

import streamlit as st
import requests
import matplotlib.pyplot as plt
import numpy as np
import json
import re
import time
from textblob import TextBlob
import plotly.graph_objects as go

# Import tooltip dictionary
from kpi_tooltips import TOOLTIPS

# ------------------------------- CONFIG -------------------------------
BACKEND_URL = "http://localhost:8080"
st.set_page_config(page_title="Voice Insights AI", layout="wide")


# ------------------------------- HELPERS -------------------------------
def safe_get(d, *path, default=None):
    cur = d
    for p in path:
        if not isinstance(cur, dict):
            return default
        cur = cur.get(p, default)
    return default if cur is None else cur


def ensure_float(val, default=0.0):
    try:
        return float(val)
    except:
        return default


# Tooltip wrapper — Option 1 (native HTML hover)
def label_with_tip(text, key):
    tip = TOOLTIPS.get(key, "")
    return f'<span title="{tip}"><b>{text}</b></span>'


# ------------------------------- DIARIZATION -------------------------------
def diarize_keep_transcript_speakers(transcript: str):
    lines = transcript.splitlines()
    diarized = []
    ts = 0.0

    pattern = re.compile(r'^(Speaker\s*\d+)\s*[:\-]\s*(.*)$', flags=re.I)

    for raw in lines:
        line = raw.strip()
        if not line:
            continue

        m = pattern.match(line)
        if not m:
            if diarized:
                diarized[-1]["text"] += " " + line
                try:
                    diarized[-1]["sentiment"] = float(TextBlob(diarized[-1]["text"]).sentiment.polarity)
                except:
                    diarized[-1]["sentiment"] = 0.0
            continue

        speaker = m.group(1)
        text = m.group(2).strip()
        try:
            sentiment = float(TextBlob(text).sentiment.polarity)
        except:
            sentiment = 0.0

        diarized.append({
            "speaker": speaker,
            "text": text,
            "timestamp": ts,
            "sentiment": sentiment
        })

        ts += float(np.random.uniform(3, 6))

    return diarized


# ------------------------------- VISUALS -------------------------------
def gauge(title, value):
    v = ensure_float(value)
    v = max(0, min(v, 1))

    fig = go.Figure(go.Indicator(
        mode="gauge+number",
        value=v,
        gauge={
            'axis': {'range': [0, 1]},
            'bar': {'color': "#2a9df4"},
        },
        title={'text': title}
    ))

    fig.update_layout(height=240, margin=dict(l=10, r=10, t=40, b=10))
    return fig


def radar_agent_quality(agent: dict):
    categories = ["Rapport", "Professionalism", "Accuracy", "Confidence"]

    def val(k):
        if isinstance(agent.get(k), str) and agent[k].endswith("%"):
            return float(agent[k].replace("%", "")) / 100
        return ensure_float(agent.get(k, 0))

    vals = [
        val("rapport_score"),
        val("professionalism_score"),
        val("solution_accuracy_score"),
        1.0 if str(agent.get("agent_confidence_level", "")).lower() in ["high", "confident"] else 0.2
    ]

    vals = [max(0, min(v, 1)) for v in vals]

    fig = go.Figure()
    fig.add_trace(go.Scatterpolar(
        r=vals + vals[:1],
        theta=categories + categories[:1],
        fill='toself'
    ))

    fig.update_layout(
        polar=dict(radialaxis=dict(visible=True, range=[0, 1])),
        showlegend=False,
        height=360
    )
    return fig


def sankey_call_flow(cp, aa, kpi, bi):
    label_issue = cp.get("primary_issue", "Customer Issue")
    label_agent = "Agent Response"
    rl = ensure_float(kpi.get("resolution_likelihood", 0))
    label_res = f"Resolution ({int(rl * 100)}%)"
    label_impact = bi.get("service_gap_identified", "Business Impact")

    labels = [label_issue, label_agent, label_res, label_impact]
    sources = [0, 1, 2]
    targets = [1, 2, 3]
    values = [
        1.0,
        rl,
        ensure_float(bi.get("risk_of_churn", 0))
    ]

    fig = go.Figure(go.Sankey(
        node=dict(label=labels),
        link=dict(source=sources, target=targets, value=values)
    ))

    fig.update_layout(height=360)
    return fig


# ------------------------------- PARSER -------------------------------
def parse_kpi_extraction(root: dict):
    return {
        "customer_problem": root.get("customer_problem", {}),
        "agent_analysis": root.get("agent_analysis", {}),
        "kpi": root.get("kpi", {}),
        "should_have_done": root.get("should_have_done", {}),
        "actions": root.get("actions", {}),
        "conversation_quality": root.get("conversation_quality", {}),
        "trend_insights": root.get("trend_insights_from_similar_calls", {}),
        "business_impact": root.get("business_impact", {}),
    }


# ------------------------------- SESSION -------------------------------
if "history" not in st.session_state:
    st.session_state.history = []


# ------------------------------- UI -------------------------------
st.title("🎙️ Voice Insights AI — Final Dashboard")
st.caption("AI-powered intelligence from your call recordings.")

audio_url = st.text_input("Enter Audio URL:")
k = st.number_input("K (number of relevant records to fetch)", min_value=1, max_value=10, value=3)

if st.button("Analyze Call"):
    if not audio_url.strip():
        st.error("Enter a valid URL.")
        st.stop()

    st.audio(audio_url)

    with st.spinner("Processing..."):
        try:
            resp = requests.get(
                f"{BACKEND_URL}/process",
                params={"audio_url": audio_url, "k": k},
                timeout=180
            )
            data = resp.json()
        except Exception as e:
            st.error(str(e))
            st.stop()

    if "error" in data:
        st.error(data["error"])
        st.json(data)
        st.stop()

    transcript = data.get("transcript", "")
    parsed = parse_kpi_extraction(data.get("kpi_extraction", {}))

    cp = parsed["customer_problem"]
    aa = parsed["agent_analysis"]
    kpi = parsed["kpi"]
    sh = parsed["should_have_done"]
    actions = parsed["actions"]
    cq = parsed["conversation_quality"]
    ti = parsed["trend_insights"]
    bi = parsed["business_impact"]
    actions = parsed["actions"]
    evidence = data.get("evidence", {})

    st.session_state.history.append(data)

    # -------------------------------- TABS --------------------------------
    tab1, tab2, tab3, tab4, tab5, tab6, tab7, tab8, tab9 = st.tabs([
        "Call Input", "Transcript & Timeline", "Customer Problem",
        "Agent Performance", "KPI Metrics", "Conversation Quality",
        "Trend Insights", "Business Impact", "Actionables"
    ])

    # ---------------- TAB 1: CALL INPUT ----------------
    with tab1:
        st.write("**Audio URL:**", audio_url)
        st.json({"duration_ms": data.get("duration_ms", 0)})

    # ---------------- TAB 2: TRANSCRIPT ----------------
    with tab2:
        diarized = diarize_keep_transcript_speakers(transcript)
        st.subheader("Diarized Transcript")

        for t in diarized:
            color = "#3A8DC9" if t["speaker"] == "Speaker 1" else "#38C744"
            st.markdown(
                f"""
                <div style='background:{color};padding:10px;border-radius:6px;margin-bottom:4px;'>
                    <b>{t['speaker']}:</b> {t['text']}
                </div>
                """,
                unsafe_allow_html=True
            )

    # ---------------- TAB 3: CUSTOMER PROBLEM ----------------
    with tab3:
        for key, val in cp.items():
            st.markdown(f"{label_with_tip(key.replace('_',' ').title(), key)}: {val}", unsafe_allow_html=True)

    # ---------------- TAB 4: AGENT PERFORMANCE ----------------
    with tab4:
        for key, val in aa.items():

            # Heading with tooltip

            if isinstance(val, list):
                st.markdown(label_with_tip(key.replace("_", " ").title(), key), unsafe_allow_html=True)
                if len(val) == 0:
                    st.markdown("&nbsp;&nbsp;&nbsp;&nbsp;• _None_<br>", unsafe_allow_html=True)
                else:
                    bullet_text = ""
                    for v in val:
                        bullet_text += f"&nbsp;&nbsp;&nbsp;&nbsp;• {v}<br>"
                    st.markdown(bullet_text, unsafe_allow_html=True)
                st.markdown("---") 

            else:
                st.markdown(f"{label_with_tip(key.replace('_',' ').title(), key)}: {val}", unsafe_allow_html=True)

        st.plotly_chart(radar_agent_quality(aa), use_container_width=True)

    # ---------------- TAB 5: KPI METRICS ----------------
    with tab5:
        st.subheader("Gauges")

        g1 = st.columns(3)
        g1[0].plotly_chart(gauge("Customer Talk Ratio", kpi.get("customer_talk_ratio", 0)))
        g1[1].plotly_chart(gauge("Agent Talk Ratio", kpi.get("agent_talk_ratio", 0)))
        g1[2].plotly_chart(gauge("Resolution", kpi.get("resolution_likelihood", 0)))

        g2 = st.columns(3)
        g2[0].plotly_chart(gauge("Frustration", kpi.get("frustration_score", 0)))
        g2[1].plotly_chart(gauge("Confusion", kpi.get("confusion_level", 0)))
        g2[2].plotly_chart(gauge("Empathy", kpi.get("empathy_score", 0)))

        # -------------------- FILTERED NUMERIC KPIs --------------------
        st.subheader("Numeric KPIs")

        # KPIs that should NOT be shown here (already in gauges)
        normalized_keys = {
            "customer_talk_ratio",
            "agent_talk_ratio",
            "frustration_score",
            "confusion_level",
            "empathy_score",
            "resolution_likelihood"
        }

        # Filter remaining numeric KPIs
        remaining_kpis = {k: v for k, v in kpi.items() if k not in normalized_keys}

        # Display 3 per row
        keys = list(remaining_kpis.keys())

        for i in range(0, len(keys), 3):
            row = st.columns(3)
            for col_idx, k in enumerate(keys[i:i+3]):
                with row[col_idx]:
                    label = label_with_tip(k.replace("_"," ").title(), k)
                    st.markdown(
                        f"{label}: <b>{remaining_kpis[k]}</b>",
                        unsafe_allow_html=True
                    )


    # ---------------- TAB 6: CONVERSATION QUALITY ----------------
    with tab6:

        st.subheader("Gauges")

        g1 = st.columns(2)
        g1[0].plotly_chart(gauge("Agent Talk Ratio", cq.get("clarity_score", 0)))
        g1[1].plotly_chart(gauge("Listening Score", cq.get("listening_score", 0)))

        g2 = st.columns(2)
        g2[0].plotly_chart(gauge("Relevance Score", cq.get("relevance_score", 0)))
        g2[1].plotly_chart(gauge("Trust Building Score", cq.get("trust_building_score", 0)))

        g3 = st.columns(1)
        g3[0].plotly_chart(gauge("Overall Score", cq.get("overall_score", 0)))

        normalized_cqs = {
            "overall_score",
            "clarity_score",
            "listening_score",
            "relevance_score",
            "trust_building_score"
        }

        remaining_cqs = {k: v for k, v in cq.items() if k not in normalized_cqs}

        for key in remaining_cqs:
            if isinstance(cq.get(key), list):
                st.markdown(label_with_tip(key.replace("_", " ").title(), key), unsafe_allow_html=True)
                if len(cq.get(key)) == 0:
                    st.markdown("&nbsp;&nbsp;&nbsp;&nbsp;• _None_<br>", unsafe_allow_html=True)
                else:
                    bullet_text = ""
                    for v in cq.get(key):
                        bullet_text += f"&nbsp;&nbsp;&nbsp;&nbsp;• {v}<br>"
                    st.markdown(bullet_text, unsafe_allow_html=True)
                st.markdown("---")
            else:
                st.markdown(f"{label_with_tip(key.replace('_',' ').title(), key)}: {cq.get(key)}", unsafe_allow_html=True)

    # ---------------- TAB 7: TREND INSIGHTS ----------------
    with tab7:
        for key, val in ti.items():
            st.markdown(f"{label_with_tip(key.replace('_',' ').title(), key)}: {val}", unsafe_allow_html=True)

    # ---------------- TAB 8: BUSINESS IMPACT ----------------
    with tab8:
        for key, val in bi.items():
            st.markdown(f"{label_with_tip(key.replace('_',' ').title(), key)}: {val}", unsafe_allow_html=True)

        st.subheader("Sankey Summary Flow")
        st.plotly_chart(sankey_call_flow(cp, aa, kpi, bi), use_container_width=True)

        st.subheader("Evidence")
        st.json(evidence)

    with tab9:
        for key, val in actions.items():

            # Heading with tooltip

            if isinstance(val, list):
                st.markdown(label_with_tip(key.replace("_", " ").title(), key), unsafe_allow_html=True)
                if len(val) == 0:
                    st.markdown("&nbsp;&nbsp;&nbsp;&nbsp;• _None_<br>", unsafe_allow_html=True)
                else:
                    bullet_text = ""
                    for v in val:
                        bullet_text += f"&nbsp;&nbsp;&nbsp;&nbsp;• {v}<br>"
                    st.markdown(bullet_text, unsafe_allow_html=True)
                st.markdown("---") 

            else:
                st.markdown(f"{label_with_tip(key.replace('_',' ').title(), key)}: {val}", unsafe_allow_html=True)




st.sidebar.markdown("---")
st.sidebar.write("Built with ❤️ @ IndiaMART")
