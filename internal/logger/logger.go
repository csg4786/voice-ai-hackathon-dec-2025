package logger

import (
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Entry
}

func New() *Logger {
	base := logrus.New()

	// Local env = pretty console; others = JSON
	env := os.Getenv("ENVIRONMENT")
	if env == "" || env == "local" {
		base.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
			ForceColors:     true,
		})
	} else {
		base.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	}

	base.SetOutput(os.Stdout)

	// Log level
	level := os.Getenv("LOG_LEVEL")
	switch level {
	case "debug":
		base.SetLevel(logrus.DebugLevel)
	case "warn":
		base.SetLevel(logrus.WarnLevel)
	case "error":
		base.SetLevel(logrus.ErrorLevel)
	default:
		base.SetLevel(logrus.InfoLevel)
	}

	return &Logger{Entry: logrus.NewEntry(base)}
}

// WithRequest attaches request metadata and returns an entry
func (l *Logger) WithRequest(r *http.Request) *logrus.Entry {
	reqID := r.Header.Get("X-Request-ID")
	if reqID == "" {
		reqID = uuid.New().String()
	}

	return l.WithFields(logrus.Fields{
		"req_id":     reqID,
		"method":     r.Method,
		"path":       r.URL.Path,
		"remote_ip":  r.RemoteAddr,
		"user_agent": r.UserAgent(),
	})
}

// WithError standardizes error logging
func (l *Logger) WithError(err error) *logrus.Entry {
	if err == nil {
		return l.Entry
	}
	return l.Entry.WithField("error", err.Error())
}
