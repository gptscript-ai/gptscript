package mvl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Minimally Viable Logger

// This Package exists because I couldn't decide on a logging framework that didn't infuriate me.
// So this is simple place to make a better decision later about logging frameworks. I only care about
// the interface, not the implementation. Smarter people do that well.

func SetSimpleFormat(trunc bool) {
	logrus.SetFormatter(&formatter{
		trunc: trunc,
	})
}

type formatter struct {
	trunc bool
}

func (f formatter) Format(entry *logrus.Entry) ([]byte, error) {
	msg := entry.Message
	if i, ok := entry.Data["input"].(string); ok && i != "" {
		msg += fmt.Sprintf(" [input=%s]", i)
	}
	if i, ok := entry.Data["output"].(string); ok && i != "" {
		if f.trunc {
			i = strings.TrimSpace(i)
			addDot := false
			if len(i) > 100 {
				addDot = true
				i = i[:100]
			}
			d, _ := json.Marshal(i)
			i = string(d)
			i = strings.TrimSpace(i[1 : len(i)-1])
			if addDot {
				i += "..."
			}
		}
		msg += fmt.Sprintf(" [output=%s]", i)
	}
	if i, ok := entry.Data["request"]; ok && i != "" {
		msg += fmt.Sprintf(" [request=%s]", i)
	}
	if i, ok := entry.Data["cached"]; ok && i == true {
		msg += " [cached]"
	}
	if i, ok := entry.Data["response"]; ok && i != "" {
		msg += fmt.Sprintf(" [response=%s]", i)
	}
	if i, ok := entry.Data["total"]; ok && i != "" {
		msg += fmt.Sprintf(" [total=%v]", i)
	}
	if i, ok := entry.Data["prompt"]; ok && i != "" {
		msg += fmt.Sprintf(" [prompt=%v]", i)
	}
	if i, ok := entry.Data["completion"]; ok && i != "" {
		msg += fmt.Sprintf(" [completion=%v]", i)
	}
	return []byte(fmt.Sprintf("%s %s\n",
		entry.Time.Format(time.TimeOnly),
		msg)), nil
}

func SetDebug() {
	logrus.SetFormatter(&logrus.JSONFormatter{
		PrettyPrint: os.Getenv("GPTSCRIPT_JSON_LOG_SINGLE_LINE") != "true",
	})
	logrus.SetLevel(logrus.DebugLevel)
}

func SetError() {
	logrus.SetLevel(logrus.ErrorLevel)
}

func Package() Logger {
	_, p, _, _ := runtime.Caller(1)
	_, suffix, _ := strings.Cut(p, "gptscript")
	i := strings.LastIndex(suffix, "/")
	if i > 0 {
		return New(suffix[:i])
	}
	return New(p)
}

func NewWithFields(fields logrus.Fields) Logger {
	return Logger{
		log:    logrus.StandardLogger(),
		fields: fields,
	}
}

func NewWithID(id string) Logger {
	return NewWithFields(logrus.Fields{
		"id": id,
	})
}

func New(name string) Logger {
	var fields logrus.Fields
	if name != "" {
		fields = logrus.Fields{
			"logger": name,
		}
	}
	return NewWithFields(fields)
}

func SetOutput(out io.Writer) {
	logrus.SetOutput(out)
}

type Logger struct {
	log    *logrus.Logger
	fields logrus.Fields
}

func (l *Logger) FieldsMap(kv map[string]any) *Logger {
	newFields := map[string]any{}
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range kv {
		newFields[k] = v
	}
	return &Logger{
		log:    l.log,
		fields: newFields,
	}
}

func (l *Logger) Fields(kv ...any) *Logger {
	newFields := map[string]any{}
	for k, v := range l.fields {
		newFields[k] = v
	}
	for i, v := range kv {
		if i%2 == 1 {
			newFields[kv[i-1].(string)] = v
		}
	}
	return &Logger{
		log:    l.log,
		fields: newFields,
	}
}

type InfoLogger interface {
	Infof(msg string, args ...any)
}

type infoKey struct{}

func WithInfo(ctx context.Context, logger InfoLogger) context.Context {
	return context.WithValue(ctx, infoKey{}, logger)
}

func (l *Logger) InfofCtx(ctx context.Context, msg string, args ...any) {
	il, ok := ctx.Value(infoKey{}).(InfoLogger)
	if ok {
		il.Infof(msg, args...)
		return
	}
	l.log.WithFields(l.fields).Infof(msg, args...)
}

func (l *Logger) Infof(msg string, args ...any) {
	l.log.WithFields(l.fields).Infof(msg, args...)
}

func (l *Logger) Errorf(msg string, args ...any) {
	l.log.WithFields(l.fields).Errorf(msg, args...)
}

func (l *Logger) Tracef(msg string, args ...any) {
	l.log.WithFields(l.fields).Tracef(msg, args...)
}

func (l *Logger) Warnf(msg string, args ...any) {
	l.log.WithFields(l.fields).Warnf(msg, args...)
}

func (l *Logger) IsDebug() bool {
	return l.log.IsLevelEnabled(logrus.DebugLevel)
}

func (l *Logger) Debugf(msg string, args ...any) {
	l.log.WithFields(l.fields).Debugf(msg, args...)
}

func (l *Logger) Fatalf(msg string, args ...any) {
	l.log.WithFields(l.fields).Fatalf(msg, args...)
}
