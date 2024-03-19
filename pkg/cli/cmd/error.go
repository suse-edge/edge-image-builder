package cmd

import (
	"go.uber.org/zap"

	"github.com/suse-edge/edge-image-builder/pkg/log"
)

type Error struct {
	UserMessage string
	LogMessage  string
}

func LogError(err *Error, checkLogMessage string) {
	if err.LogMessage == "" {
		log.AuditError(err.UserMessage)
		return
	}

	log.Audit(err.UserMessage)
	log.Audit(checkLogMessage)
	zap.S().Error(err.LogMessage)
}
