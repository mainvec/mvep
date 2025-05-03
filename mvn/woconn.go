package won

import (
	"errors"
	"net/url"
	"sort"
	"sync"
	"time"
)

type WOContextHandlerFunc func(data any) (any, error)

type WOConn interface {
	// Close()
	// GetNativeConn() interface{}
	// QueueSubscribe(subject string, queue string, wocmd interface{}, hndlr WOContextHandlerFunc)
	// Request(subj string, data []byte, timeout time.Duration) (WOMsg, error)
	// PingSubscribe(pingSubject string)

	//Pings connected node. return true is successful
	Ping() (bool, error)
	Request(subj string, data []byte, timeout time.Duration) ([]byte, error)
	Close() error
	PingSubscribe(pingSubject string)
	Subscribe(subj string, hf func(data []byte) ([]byte, error)) error
	QueueSubscribe(subject string, queue string, hf func(data []byte) ([]byte, error)) error
}

type WOConnDriver interface {
	GetPrefix() string
	NewWOConn(wourl *url.URL) (WOConn, error)
}

var (
	wodConnDriversMu sync.RWMutex
	woConnDrivers    = make(map[string]WOConnDriver)
)

// makes a woconn driver available by the provided prefix.
// If Register is called twice with the same name or if driver is nil,
// it panics. (Excatly as in GO SQL)
func RegisterWOConnDriver(driver WOConnDriver) {
	wodConnDriversMu.Lock()
	defer wodConnDriversMu.Unlock()
	name := driver.GetPrefix()
	if driver == nil {
		panic("WOConnDriver: Register driver is nil")
	}
	if _, dup := woConnDrivers[name]; dup {
		panic("WOConnDriver: Register called twice for driver " + name)
	}
	woConnDrivers[name] = driver
}

// Drivers returns a sorted list of the names of the registered drivers.
func WOConnDrivers() []string {
	wodConnDriversMu.RLock()
	defer wodConnDriversMu.RUnlock()
	list := make([]string, 0, len(woConnDrivers))
	for name := range woConnDrivers {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

func Dial(conn_url string) (WOConn, error) {
	u, err := url.Parse(conn_url)
	if err != nil {
		return nil, err
	}
	scheme := u.Scheme
	wodriver, ok := woConnDrivers[scheme]
	if !ok {
		return nil, errors.New("unknown WOConn scheme")
	}
	return wodriver.NewWOConn(u)
}
