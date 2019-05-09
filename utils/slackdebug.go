package utils

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

// Provides a Slack way to send fmt.Println log to Slack for debugging purpose
// Please Disable this in Production otherwise, it would be very slow.
// Since doing docker logs storage-node_app_1 is very tedious and slack is very responsive.
// This is better way to view the log.

// In order to work you. You need to create a slack channel and setup webhook for
// debugging

type LogLevel int

const (
	Info LogLevel = iota + 1
	Warn
	Error
)

func SlackLog(message string) {
	SlackLogWithLevel(message, Info)
}

func SlackLogError(message string) {
	SlackLogWithLevel(message, Error)
}

func SlackLogWithLevel(message string, level LogLevel) {
	GetDefaultLogger().Info(message)

	if len(Env.SlackDebugUrl) == 0 {
		return
	}

	attachment := map[string]string{
		"color": getLogLevelColor(level),
		"text":  Env.DisplayName + ": " + message,
	}

	values := map[string]interface{}{
		"attachments": []map[string]string{attachment},
		"username":    Env.DisplayName,
		"icon_emoji":  ":ghost:",
	}

	jsonValue, _ := json.Marshal(values)
	_, err := http.Post(strings.TrimSpace(Env.SlackDebugUrl), "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		GetDefaultLogger().Errorf("Unable to Post to Slack due to %v", err)
	}
}

func getLogLevelColor(level LogLevel) string {
	switch level {
	case Info:
		return "good"
	case Warn:
		return "warning"
	case Error:
		return "danger"
	}
	// Gray HEX color
	return "#bababa"
}
