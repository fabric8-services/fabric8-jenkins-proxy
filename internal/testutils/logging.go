package testutils

import (
	log "github.com/sirupsen/logrus"
)

// ExtractLogMessages extracts log messages from an array of *log.Entry and return an array of message strings.
func ExtractLogMessages(entries []*log.Entry) []string {
	messages := []string{}
	for _, logEntry := range entries {
		messages = append(messages, logEntry.Message)
	}
	return messages
}
