package transportc

import "errors"

var ErrDatachannelNotReady = errors.New("Data Channel is not ready")
var ErrDataChannelClosed = errors.New("Data Channel is closed")
var ErrDataChannelAtCapacity = errors.New("Data Channel is at capacity")
