package types

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.Logger
	sugared *zap.SugaredLogger

	LogsPath   string
	Name       string
	FileWriter *os.File
}

func (l *Logger) Sugared() *zap.SugaredLogger {
	if l.sugared == nil {
		l.sugared = l.Sugar()
	}
	return l.sugared
}

func (l *Logger) Debugf(template string, args ...interface{}) {
	l.Sugared().Debugf(template, args...)
}

func (l *Logger) Infof(template string, args ...interface{}) {
	l.Sugared().Infof(template, args...)
}

func (l *Logger) Warnf(template string, args ...interface{}) {
	l.Sugared().Warnf(template, args...)
}

func (l *Logger) Errorf(template string, args ...interface{}) {
	l.Sugared().Errorf(template, args...)
}

func (l *Logger) Fatalf(template string, args ...interface{}) {
	l.Sugared().Fatalf(template, args...)
}

func (l *Logger) Panicf(template string, args ...interface{}) {
	l.Sugared().Panicf(template, args...)
}

func (l *Logger) DPanicf(template string, args ...interface{}) {
	l.Sugared().DPanicf(template, args...)
}

func (l *Logger) Debugw(msg string, keysAndValues ...interface{}) {
	l.Sugared().Debugw(msg, keysAndValues...)
}

func (l *Logger) Infow(msg string, keysAndValues ...interface{}) {
	l.Sugared().Infow(msg, keysAndValues...)
}

func (l *Logger) Warnw(msg string, keysAndValues ...interface{}) {
	l.Sugared().Warnw(msg, keysAndValues...)
}

func (l *Logger) Errorw(msg string, keysAndValues ...interface{}) {
	l.Sugared().Errorw(msg, keysAndValues...)
}

func (l *Logger) Fatalw(msg string, keysAndValues ...interface{}) {
	l.Sugared().Fatalw(msg, keysAndValues...)
}

func (l *Logger) Panicw(msg string, keysAndValues ...interface{}) {
	l.Sugared().Panicw(msg, keysAndValues...)
}

func (l *Logger) DPanicw(msg string, keysAndValues ...interface{}) {
	l.Sugared().DPanicw(msg, keysAndValues...)
}

type Log struct {
	Timestamp  time.Time
	Caller     string
	LoggerName string
	Level      zapcore.Level
	Message    string
}

type LogHook func(log Log)
