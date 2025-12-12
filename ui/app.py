import streamlit as st
import requests
import json

BACKEND_URL = "http://localhost:8080"   # change to Cloud Run URL later

st.set_page_config(page_title="Voice Insights AI", layout="wide")

st.title("🎙️ Voice Insights AI – Call Understanding Engine")

st.markdown("""
Paste **any call recording URL** below.  
This interface will:
1. Send the URL to the Go backend  
2. Backend transcribes it using the IM transcription API  
3. LLM analyzes dataset patterns + transcript  
4. Returns actionable insights for support / onboarding / pricing  
""")

audio_url = st.text_input("Call Recording URL", placeholder="https://.../call.wav")

if st.button("Process"):
    if not audio_url.strip():
        st.error("Please provide a valid audio URL.")
    else:
        with st.spinner("Transcribing & Analyzing..."):
            try:
                resp = requests.get(
                    f"{BACKEND_URL}/process",
                    params={"audio_url": audio_url},
                    timeout=120
                )
                data = resp.json()
            except Exception as e:
                st.error(f"Request failed: {e}")
                st.stop()

        if "error" in data and data["error"]:
            st.error("Backend Error: " + data["error"])
            st.json(data)
            st.stop()

        st.success("Analysis Complete! 🎉")

        # Display transcript
        with st.expander("📄 Transcript"):
            st.write(data.get("transcript", "<empty>"))

        # Display extraction fields
        extraction = data.get("extraction", {})
        st.subheader("🔍 Insights")
        col1, col2, col3 = st.columns(3)
        col1.metric("Category", extraction.get("category", ""))
        col2.metric("Sentiment", extraction.get("sentiment", ""))
        col3.metric("Confused?", "Yes" if extraction.get("is_confused") else "No")

        st.markdown(f"**Root Cause:** {extraction.get('root_cause', '')}")
        st.markdown(f"**Escalation Reason:** {extraction.get('escalation_reason', '')}")

        # Action card
        st.subheader("🟦 Recommended Action")
        ac = data.get("action_card", {})
        st.markdown(f"**Insight:** {ac.get('insight', '')}")
        st.markdown(f"**Action:** {ac.get('action', '')}")
        st.markdown(f"**Impact:** {ac.get('impact', '')}")

        # Dataset evidence
        st.subheader("📊 Dataset Evidence")
        st.json(data.get("evidence", {}))

st.markdown("---")
st.caption("Powered by IndiaMART Voice AI + LLM Gateway")
