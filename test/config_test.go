package transportc_test

import (
	"github.com/gaukas/transportc"
)

var defaultConfig = &transportc.Config{
	SignalMethod: transportc.NewDebugSignal(3),
}

func getDefaultDialer() (*transportc.Dialer, error) {
	return defaultConfig.NewDialer()
}

func getDefaultListener() (*transportc.Listener, error) {
	return defaultConfig.NewListener()
}
