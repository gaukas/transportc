package transportc_test

import (
	"github.com/Gaukas/transportc"
)

var defaultConfig = &transportc.Config{}

func getDefaultDialer() (*transportc.Dialer, error) {
	return defaultConfig.NewDialer(nil)
}

func getDefaultListener() (*transportc.Listener, error) {
	return defaultConfig.NewListener(nil)
}
