package comm

import (
	"net"
	"sync"
)

// Connection responder that manages the net.Conn created by the server.
// To be used throughout message brokerage so no routing components need to own
// the conn object.
type ConnResponder struct {
	C  net.Conn
	mu sync.Mutex
}

func NewConnResponder(conn net.Conn) *ConnResponder {
	return &ConnResponder{
		C: conn,
	}
}

func (cr *ConnResponder) RemoteAddr() string {
	return cr.C.RemoteAddr().String()
}

// Send the given payload back to the connection's return address.
func (cr *ConnResponder) Write(b []byte) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	_, err := cr.C.Write(b)
	return err
}
