package config

import (
	"time"

	"github.com/spf13/viper"
)

type LoggerConfig interface {
	Debug() bool
	LogToFile() bool
	LogsDir() string
	Timezone() *time.Location
}

type loggerConfig struct {
	debug     bool
	logToFile bool
	logsDir   string
	timezone  *time.Location
}

func NewLoggerConfig() LoggerConfig {
	timezone, _ := time.LoadLocation(viper.GetString("logger.timezone"))
	return &loggerConfig{
		debug:     viper.GetBool("logger.debug"),
		logToFile: viper.GetBool("logger.logs-to-file"),
		logsDir:   viper.GetString("logger.logs-dir"),
		timezone:  timezone,
	}
}

func (l *loggerConfig) Debug() bool              { return l.debug }
func (l *loggerConfig) LogToFile() bool          { return l.logToFile }
func (l *loggerConfig) LogsDir() string          { return l.logsDir }
func (l *loggerConfig) Timezone() *time.Location { return l.timezone }
