package testutils

import (
	log "github.com/sirupsen/logrus"
)

func ExtractLogMessages(entries []*log.Entry) []string {
	messages := []string{}
	for _, logEntry := range entries {
		messages = append(messages, logEntry.Message)
	}
	return messages
}
