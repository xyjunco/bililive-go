package listeners

import (
	"github.com/xyjunco/bililive-go/src/lib/events"
)

const (
	ListenStart events.EventType = "ListenStart"
	ListenStop  events.EventType = "ListenStop"
	LiveStart   events.EventType = "LiveStart"
	LiveEnd     events.EventType = "LiveEnd"
)
