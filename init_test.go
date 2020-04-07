package eventlistener

import (
	"os"
)

func init() {
	if os.Getenv("DEBUG") != "" {
		EnableDebugLogging()
	}
}
