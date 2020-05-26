package eventlistener

import "flag"

var token = flag.String("token", "83e81fbcb975e4f3ce8ddf28a99e01e738154be3ff09308a89b52cbd289594a9", "user token")
var addr = flag.String("addr", ":8888", "action monitor service address")

func init() {
	flag.Parse()
	EnableDebugLogging()
}
