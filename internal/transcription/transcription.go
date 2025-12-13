package transcription

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"voice-insights-go/internal/logger"
)

var httpClient = &http.Client{Timeout: 60 * time.Second}

type PublishSuccessResponse struct {
	Code   int    `json:"Code"`
	Status string `json:"Status"`
	Data   struct {
		MediaId          string `json:"MediaId"`
		Status           string `json:"Status"`
		LanguageId       int    `json:"LanguageId"`
		TranscriptionURL string `json:"TranscriptionURL"`
		WordsCount       int    `json:"WordsCount"`
	} `json:"Data"`
	Reason   string `json:"Reason,omitempty"`
	UniqueId string `json:"UniqueId,omitempty"`
}

type StatusResponse struct {
	Code   int    `json:"Code"`
	Status string `json:"Status"`
	Data   struct {
		AudioURL             string `json:"AudioURL"`
		LanguageId           int    `json:"LanguageId"`
		Status               string `json:"Status"`
		TranscriptionTextURL string `json:"TranscriptionTextURL"`
		WordsCount           int    `json:"WordsCount"`
	} `json:"Data"`
	Reason   string `json:"Reason,omitempty"`
	UniqueId string `json:"UniqueId,omitempty"`
}

// GetTranscript: top-level call. Supports mock mode via env USE_MOCK_TRANSCRIBE=true
func GetTranscript(callURL string) (string, error) {
	log := logger.New().WithField("component", "transcription").WithField("call_url", callURL)
	if os.Getenv("USE_MOCK_TRANSCRIBE") == "true" {
		log.Info("USE_MOCK_TRANSCRIBE=true, returning mock transcript")
		return "Speaker 1: Hello.\nSpeaker 2: Hello.\nSpeaker 1: Haan bolिए. Sir, aapne check keye Lead?\nSpeaker 1: Haan bolिए bolिए.\nSpeaker 2: Aapne bail list check ki?\nSpeaker 1: Kya kare?\nSpeaker 2: Aapne bail list check ki?\nSpeaker 1: Haan check to kiya.\nSpeaker 2: Accha ek minit ruko. Maine bhi kuch leads check ki thi.\nSpeaker 1: Accha.\nSpeaker 2: Aap abhi laptop par hai?\nSpeaker 1: Haan hai.\nSpeaker 2: Accha. To maine aapko ek video diya hai video meet ka.\nSpeaker 1: Hello. Haan bolिए.\nSpeaker 2: Sir, maine aapko ek link diya hai video meet ka. Maine bhi aapki kuch leads check kari thi category wise.\nSpeaker 1: Okay.\nSpeaker 2: To main aapko dikha deti hu screen share kar ke. Ek baar meeting join kar lijiye apna.\nSpeaker 1: Haan, computer mein hai abhi bolie.\nSpeaker 2: Ek baar meeting join kijiye apna mail kholiye.\nSpeaker 1: Mail, mail khul raha hai. Aap khul diye?\nSpeaker 2: Meeting join kar lijiye.\nSpeaker 1: Haan bolie.\nSpeaker 2: Aaya hai mail video meet ka?\nSpeaker 1: Haan, aaya hai.\nSpeaker 2: Aap usko join kijiye.\nSpeaker 1: Haan kar raha hu.\nSpeaker 1: Aa gaya.\nSpeaker 2: Sir, dikh rahi hai main aapko category mein.\nSpeaker 1: Haan bolie.\nSpeaker 2: Main apni screen share karu aap. Haan bolie.\nSpeaker 1: Okay.\nSpeaker 2: Haan. Haan bolie. Ji sir, ek minute bas.\nSpeaker 1: Haan.\nSpeaker 2: Theek hai, sir. Yahan par hum aa gaye category report mein. Theek hai? Yeh aapki categories hai. Theek hai? Sabse pehle hum dekh lete hai GST return filing. Theek hai? Abhi GST return filing service mein yeh lead available hai. Theek hai.\nSpeaker 1: Koi lead nahi hai.\nSpeaker 2: Theek hai, yeh aap West Bengal mein hi karte ho. GST return filing. Ki all India bataya tha aap ne? Ki all India.\nSpeaker 1: All India. All India. All India.\nSpeaker 2: To yeh kar sakte ho?\nSpeaker 1: Haan kar sakte hai to, koi kahan par hai, lead ka par hai?\nSpeaker 2: Tamil Nadu, Chennai.\nSpeaker 1: Haan, isme madam income tax wala hai to hum kar sakte hai.\nSpeaker 2: Accha. Theek hai. Ek lead to yeh ho gayi. Ek minut ruko main dhundungi aapko iska screenshot de deti hu. Screenshot nahi hu link hi de deti hu.\nSpeaker 1: Okay.\nSpeaker 2: Aapka WhatsApp number kaun sa hai?\nSpeaker 1: 907.\nSpeaker 2: Yeh hi hai na?\nSpeaker 1: Haan. Haan yahi hai.\nSpeaker 2: Theek hai, ek yeh ho gayi. Haan. Ab isme ek aur dekh lete hai. Fire NOC service. Theek hai.\nSpeaker 1: Nhi vo hum to nahi karte, nahi karte, iska koi kaam nahi hai.\nSpeaker 2: Abhi aapne laga to rakhi hai.\nSpeaker 1: Nhi nhi, usko delete kardijiye. Humne nahi laga, wo dusra laga hai usko delete kardijiye.\nSpeaker 2: Yeh wali? Delete karne ka option.\nSpeaker 1: Haan, delete kardijiye, delete kijiye.\nSpeaker 2: Sir, main inactive kar deti hu. Delete actually aapko karna padega. Inactive kar diya hai maine to aap yahan jayenge to yeh aapko dikh jayegi. Theek hai?\nSpeaker 1: Okay.\nSpeaker 2: Ab yeh hatt jayegi. Ek minute isko main refresh karungi. Yahan se hatt chuki hogi. Theek hai?\nSpeaker 1: Okay.\nSpeaker 2: Ab dekho agar registration service. Theek hai? Abhi yeh all India mein kar sakte ho. Ya West Bengal.\nSpeaker 1: Nhi, yeh Delhi ka nahi hoga.\nSpeaker 2: Delhi ka nahi hoga na?\nSpeaker 1: Nhi nhi, nahi hoga.\nSpeaker 2: Theek hai, yeh wali hataa do. Ab dekh lo, ROC compliance. bilkul. Aur bhi compliances mein hai, DGFT consultant.\nSpeaker 1: Yeh kar sakte hai.\nSpeaker 2: Or, business and consultancy service.\nSpeaker 1: Iska kaam hai kya? Dekhna padega agar mera type ka hai to kar degi. Find management provident fund consultant. Theek hai, kar dega kar dega.\nSpeaker 2: Factory Act. Iske awaj nhi aayi.\nSpeaker 1: FSSAI License. Yeh labor law wala kar sakte hai.\nSpeaker 1: Nhi nhi nhi.\nSpeaker 2: Theek hai. IP protection service. GST and pen registration. Yeh kar sakte ho.\nSpeaker 1: Haan, agar karega GST registration to hum kar sakte hai.\nSpeaker 2: Yeah, GST or pen wale mein to hai hi thodi zyaada.\nSpeaker 1: Haan.\nSpeaker 2: Dekho yahan bhi more options aa jate hai na, yahan pe category report mein jaoge.\nSpeaker 1: Okay.\nSpeaker 2: Aapki categories dikhti hai. Theek hai?\nSpeaker 1: Accha.\nSpeaker 2: Aap jaise yahan par search kar lo compliance service. Isme aapko yeh dikh jayegi. Yeh kuch leads available hai. Trust property registration. To yeh kuch dikh jayegi.\nSpeaker 1: Nhi, Tamil Nadu ka property registration to aise nahi hoga.\nSpeaker 2: Accha. West Bengal ka hi hoga. Aur maine to lagaya hai West Bengal ka.\nSpeaker 1: Nhi, nahi aaya, nahi aaya. Koi baat nahi.\nSpeaker 2: Gaming law ki to ek lead kari thi unhone. Licensing service.\nSpeaker 1: Accha.\nSpeaker 2: Thoda sa na matlab isme aise search kar ke dhundna padega.\nSpeaker 1: Haan.\nSpeaker 2: Baaki, agar aap hafte mein agar saat aath bhi lead karoge to bhi aapka business generate to ho sakta hai.\nSpeaker 1: Accha.\nSpeaker 2: Yeh West Bengal ki aapki. Gain.\nSpeaker 1: Detergent formulation. Yeh hum nahi karte. Toh dusra kaam hai. Detergent formulation. Yeh kaam kaise karega? Quality consulting service. Cooking and power service. Toh service ko dikhaye. Kuch to kaam hai? Theek hai. Driving license nahi karte. Are wo Business consultant hai na usme sab aa jata hai na isliye aa rahi hai.\nSpeaker 1: Accha.\nSpeaker 2: Trademark registration. Copyright registration.\nSpeaker 1: MSME registration. MSME mein aap bahar kar sakte hai Bengal ki?\nSpeaker 1: Nhi, West Bengal ka kar sakte hai lekin bahar se kar sakte hai lekin woh log ki karega, dekhna padega.\nSpeaker 2: Accha. Ek to apne Madhya Pradesh ka hai, ek to Rajasthan ka hai.\nSpeaker 1: Theek hai, hum dekh lete hai. Usko ek baar dekh lete hai.\nSpeaker 2: Dusra Ek aur priority.\nSpeaker 1: Yeh bait list mein to nahi aa raha hai. Bait list mein ja kar to kuch bhi nahi aa raha hai bait list mein.\nSpeaker 2: Main bata rahi hu aise nahi aa raha hai. Is normally is ja ke agar aap particular A, B, C, D category humko daal ke search karna padega. Kya hai sir, aapki jo service hai na, wo matlab thodi si alag hai. Samjh rahe ho na?\nSpeaker 1: Hmm hmm.\nSpeaker 2: NGO registration service. So hum yahan par search kar re hai. West. West Bengal based hoga.\nSpeaker 1: Haan.\nSpeaker 2: Accha. Theek hai. Tax compliance.\nSpeaker 1: Yeh kar sakte ho. Return filing.\nSpeaker 1: Okay.\nSpeaker 1: Auditing.\nSpeaker 2: Aapke bhi option aa raha hai, yeh wala more option mein ja ke category report ka?\nSpeaker 1: Ek baar mere ko dekhna padega. Hum dekh lete hai ab isko ek baar. Pura ka pura.\nSpeaker 2: Yahan pe jaoge more options, category report.\nSpeaker 1: Okay, okay.\nSpeaker 2: To sir abhi maine kam se kam aapko itni to bata di hai ki jaise aaj aapka Friday hai. Hmm. Sunday.\nSpeaker 1: Hmm.\nSpeaker 2: Uske liye लायक lead to aapne matlab itni dikhayi hai maine aapko bhi aap consume kar sakte ho. Kyunki aap baat bhi karoge na unse. matlab phir consume karna hai na? Unko call karo, baat karoge. Phir wo deal convert hogi ki nahi hogi. Aisa hai na?\nSpeaker 1: Hmm. Theek hai. Hum baat kar, ab jis jis se baat ho sakte hai, hum baat karte hai. Theek hai.\nSpeaker 2: Theek hai. Sir, abhi aapka concern main close kar du?\nSpeaker 1: Nahi, ek baar dekh leta hu pehle. Theek hai? Ek baar dekh leta hu.\nSpeaker 2: But maine aapko leads to show kari hai na?\nSpeaker 1: Haan show kar raha hai. Ek baar dekh leta hu. Kaam ka hai ki kya se kya hai. Ek baar dekh leta hu pehle. Theek hai? Main aapko dekh ke bata dunga. Theek hai.\nSpeaker 2: To sir, aap mera number to note kar liya hai na?\nSpeaker 1: Haan haan. Number hai aapke paas. Main aapko call back kar raha hu. Ek baar dekh ke call back kar raha hu. Theek hai.\nSpeaker 2: Theek hai. Yeh option aapko abhi dikh raha hai?\nSpeaker 1: Haan.\nSpeaker 1: Theek hai. To mujhe aap. Theek hai. Main dekh leta hu, dekh leta hu haan. Kyuki woh complaint padi hui hai na isliye bolti hu.\nSpeaker 1: Okay. Theek hai, theek hai.\nSpeaker 1: Theek hai.", nil
	}
	apiHost := os.Getenv("TRANSCRIBE_URL")
	if apiHost == "" {
		log.Error("TRANSCRIBE_URL not set")
		return "", errors.New("TRANSCRIBE_URL not set")
	}
	log.Info("publishing to transcription API", apiHost)
	mediaID, existingURL, err := publish(callURL, apiHost)
	if err != nil {
		log.WithError(err).Error("publish failed")
		return "", err
	}
	if existingURL != "" {
		log.WithField("transcription_url", existingURL).Info("transcription already exists; downloading")
		return download(existingURL)
	}
	finalURL, err := poll(mediaID, apiHost)
	if err != nil {
		log.WithError(err).Error("poll failed")
		return "", err
	}
	log.WithField("final_url", finalURL).Info("download final transcript")
	return download(finalURL)
}

func publish(callURL, host string) (string, string, error) {
	log := logger.New().WithField("component", "transcription.publish").WithField("call_url", callURL)
	endpoint := strings.TrimRight(host, "/") + "/transcribe"
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	_ = w.WriteField("callRecordingLink", callURL)
	_ = w.WriteField("callType", "PNS")
	_ = w.Close()

	req, _ := http.NewRequest("POST", endpoint, &b)
	req.Header.Set("Content-Type", w.FormDataContentType())

	var resp PublishSuccessResponse
	if err := doJSON(req, &resp); err != nil {
		log.WithError(err).Error("publish request failed")
		return "", "", err
	}
	log.WithField("resp_code", resp.Code).WithField("resp_status", resp.Status).WithField("data", resp.Data).Info("publish response")
	if resp.Code != 200 {
		return "", "", fmt.Errorf("transcribe publish error: code=%d reason=%s", resp.Code, resp.Reason)
	}
	if resp.Data.TranscriptionURL != "" && strings.ToLower(resp.Data.Status) == "success" {
		return "", resp.Data.TranscriptionURL, nil
	}
	return resp.Data.MediaId, "", nil
}

func poll(mediaID, host string) (string, error) {
	log := logger.New().WithField("component", "transcription.poll").WithField("media_id", mediaID)
	base := strings.TrimRight(host, "/") + "/getstatus"
	for i := 0; i < 60; i++ {
		time.Sleep(1500 * time.Millisecond)
		u, _ := url.Parse(base)
		q := u.Query()
		q.Set("mediaId", mediaID)
		u.RawQuery = q.Encode()
		req, _ := http.NewRequest("GET", u.String(), nil)
		var s StatusResponse
		if err := doJSON(req, &s); err != nil {
			log.WithError(err).Warnf("status request failed attempt=%d", i)
			continue
		}
		log.WithField("status", s.Data.Status).WithField("words", s.Data.WordsCount).Info("status check")
		switch strings.ToLower(s.Data.Status) {
		case "success":
			return s.Data.TranscriptionTextURL, nil
		case "queued", "processing":
			continue
		case "failed":
			return "", fmt.Errorf("transcription failed: %s", s.Reason)
		}
	}
	return "", fmt.Errorf("transcription timeout")
}

func download(url string) (string, error) {
	log := logger.New().WithField("component", "transcription.download").WithField("url", url)
	resp, err := httpClient.Get(url)
	if err != nil {
		log.WithError(err).Error("failed to download transcript")
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		log.WithField("status", resp.StatusCode).WithField("body", string(b)).Error("download returned error")
		return "", fmt.Errorf("download failed: %s", string(b))
	}
	b, _ := io.ReadAll(resp.Body)
	txt := string(b)
	log.WithField("size_bytes", len(b)).Info("download complete; transcript")
	// optionally truncate log preview to avoid overwhelming console, but full content still logged because you requested full logging
	log.WithField("transcript_full", txt).Debug("transcript content")
	return txt, nil
}

func doJSON(req *http.Request, target interface{}) error {
	log := logger.New().WithField("component", "transcription.http")
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 20 * time.Second
	var lastErr error
	op := func() error {
		log.WithField("method", req.Method).WithField("url", req.URL.String()).Info("calling external transcription API")
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			log.WithError(err).Warn("http request error")
			return err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		log.WithField("status", resp.StatusCode).WithField("body_len", len(body)).Debug("raw response body")
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %s", string(body))
			return lastErr
		}
		if len(body) == 0 {
			lastErr = fmt.Errorf("empty body")
			return lastErr
		}
		if err := json.Unmarshal(body, target); err != nil {
			lastErr = fmt.Errorf("json decode error: %v body=%s", err, string(body))
			log.WithError(lastErr).Warn("json decode failed")
			return lastErr
		}
		return nil
	}
	if err := backoff.Retry(op, bo); err != nil {
		return lastErr
	}
	return nil
}
