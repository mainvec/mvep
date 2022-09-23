package won

import "errors"

const WOSYS_NODE_PING_SUB = "$wosys.wonode.ping"

var (
	ErrConnectionClosed = errors.New("connection is already closed")
)
