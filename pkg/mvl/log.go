package mvl

import (
	"runtime"

	"github.com/sirupsen/logrus"
)

// This Package exists because I couldn't decide on a logging framework that didn't infuriate me.
// So this is simple place to make a better decision later about logging frameworks. I only care about
// the interface, not the implementation. Smarter people do that well.

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func Package() Logger {
	_, p, _, _ := runtime.Caller(1)
	return New(p)
}

func New(name string) Logger {
	return Logger{
		prefix: name + ": ",
		log:    logrus.StandardLogger(),
		fields: logrus.Fields{},
	}
}

type Logger struct {
	prefix string
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
		prefix: l.prefix,
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
		prefix: l.prefix,
		log:    l.log,
		fields: newFields,
	}
}

func (l *Logger) Infof(msg string, args ...any) {
	l.log.WithFields(l.fields).Infof(l.prefix+msg, args...)
}

func (l *Logger) Errorf(msg string, args ...any) {
	l.log.WithFields(l.fields).Errorf(l.prefix+msg, args...)
}

func (l *Logger) Tracef(msg string, args ...any) {
	l.log.WithFields(l.fields).Tracef(l.prefix+msg, args...)
}

func (l *Logger) Debugf(msg string, args ...any) {
	l.log.WithFields(l.fields).Debugf(l.prefix+msg, args...)
}

func (l *Logger) Fatalf(msg string, args ...any) {
	l.log.WithFields(l.fields).Fatalf(l.prefix+msg, args...)
}
