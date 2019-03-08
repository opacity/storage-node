package utils

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// Use this interface throughout brokernode to do logging.
// A slim down copy of logrus.FieldLogger from https://github.com/sirupsen/logrus/blob/master/logrus.go#L139
type Logger interface {
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Errorf(string, ...interface{})
	Fatalf(string, ...interface{})
	Debug(...interface{})
	Info(...interface{})
	Warn(...interface{})
	Error(...interface{})
	Fatal(...interface{})
	Panic(...interface{})

	// Extends for Opacity
	LogIfError(error, map[string]interface{})
}

type loggerWrapper struct {
	logrus.FieldLogger
}

var (
	defaultLogger Logger
	testLogger    Logger
)

func (l loggerWrapper) LogIfError(err error, extraInfo map[string]interface{}) {
	if err == nil {
		return
	}

	l.Error(fmt.Sprintf("Error: %s, extra: %s", err, extraInfo))
}

// Create a new Logger from a particular requestIdPrefix.
func GetLogger(sessionId string) Logger {
	return loggerWrapper{logrus.WithFields(logrus.Fields{"session_id": sessionId})}
}

func getLogger(loggerTag string) Logger {
	return loggerWrapper{logrus.WithFields(logrus.Fields{"tag": loggerTag})}
}

func GetDefaultLogger() Logger {
	if defaultLogger == nil {
		defaultLogger = getLogger("DEFAULT")
	}
	return defaultLogger
}

// Used for Unit tests
func GetLoggerForTest() Logger {
	if testLogger == nil {
		testLogger = getLogger("TEST")
	}
	return testLogger
}
