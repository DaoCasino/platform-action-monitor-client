package eventlistener

import (
	"go.uber.org/zap"
	"os"
)

func init() {
	if os.Getenv("DEBUG") != "" {
		logger, _ := zap.NewDevelopment()
		EnableDebugLogging(logger)
	}
}
