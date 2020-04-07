package eventlistener

import "go.uber.org/zap"

var messageLog = zap.NewNop()
var pumpsLog = zap.NewNop()

func EnableDebugLogging(l *zap.Logger) {
	messageLog = l
	pumpsLog = l
}
