package eventlistener

import "go.uber.org/zap"

var messageLog = zap.NewNop()
var pumpsLog = zap.NewNop()

func EnableDebugLogging() {
	logger, _ := zap.NewDevelopment()
	SetDebugLogger(logger)
}

func SetDebugLogger(l *zap.Logger) {
	messageLog = l
	pumpsLog = l
}
