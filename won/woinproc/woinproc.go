package woinproc

/*
InProcess connections with no dependency on external node.
All subscription and routing handling by the package itself
and share by all connection. This is to be used in simple single process/thread
apps, e.g. CLI only WO apps
*/
import (
	"errors"
	"net/url"
	"sync"
	"time"

	"github.com/workoak/wop/won"
)

// Compile time validation that our types implement the expected interfaces
var (
	_        won.WOConnDriver = (*WOInProcConnDriver)(nil)
	_        won.WOConn       = (*WOInProcConn)(nil)
	woHubsMu sync.RWMutex
	woHubs   = make(map[string]*Hub)
)

func init() {
	won.RegisterWOConnDriver(&WOInProcConnDriver{})
}

type WOInProcConnDriver struct {
}

func (inprocDrv *WOInProcConnDriver) GetPrefix() string {
	return "wo-inproc"
}
func (inprocDrv *WOInProcConnDriver) NewWOConn(wourl *url.URL) (won.WOConn, error) {
	if wourl.Scheme != inprocDrv.GetPrefix() {
		return nil, errors.New("invalid woconn url for this driver")
	}
	woHubsMu.Lock()
	defer woHubsMu.Unlock()
	wohub, ok := woHubs[wourl.String()]
	newHub := false
	if !ok {
		wohub = New()
		woHubs[wourl.String()] = wohub
		newHub = true

	}

	woinprocconn := &WOInProcConn{
		WOHub: *wohub,
	}
	if newHub {
		//Standard Node ping required
		woinprocconn.PingSubscribe(won.WOSYS_NODE_PING_SUB)
	}

	return woinprocconn, nil
}

type WOInProcConn struct {
	WOHub  Hub
	closed bool
}

func (woconn *WOInProcConn) Ping() (bool, error) {
	result, err := woconn.Request(won.WOSYS_NODE_PING_SUB, []byte("PING"), time.Second)
	if err != nil {
		return false, err
	}
	if result == nil {
		return false, errors.New("no result returned for ping")
	}
	return string(result) == "PONG", nil
}

func (woconn *WOInProcConn) Request(subj string, data []byte, timeout time.Duration) ([]byte, error) {
	if woconn.closed {
		return nil, errors.New("connection is closed")
	}
	msg := Message{
		Name: subj,
		Body: data,
	}
	respChan := make(chan []byte, 1)
	msg.respCh = respChan
	published := woconn.WOHub.Publish(msg)
	//A quick and dirty way to implement req-resp
	if published {
		respData := <-respChan
		// respMsg := &Message{Name: subj + "_resp",
		// 	Body: respData,
		// }
		return respData, nil
	}
	//no subscribers founds,
	//TODO, should we return errors?
	return nil, nil
}

func (woconn *WOInProcConn) Close() error {
	if woconn.closed {
		return errors.New("connection already closed")
	}
	woconn.closed = true
	return nil
}

func (woconn *WOInProcConn) GetNativeConn() interface{} {
	return woconn
}

func (woconn *WOInProcConn) PingSubscribe(pingSubject string) {

	sub := woconn.WOHub.Subscribe(0, pingSubject)
	go func(s Subscription) {
		for msg := range s.Receiver {
			if msg.respCh != nil {
				msg.respCh <- []byte("PONG")
				close(msg.respCh)
			}
		}
	}(sub)
}

func (woconn *WOInProcConn) Subscribe(subj string, hfunc func(data []byte) ([]byte, error)) error {
	return nil
}

func (woconn *WOInProcConn) QueueSubscribe(subject string, queue string, hf func(data []byte) ([]byte, error)) error {
	return nil
}
