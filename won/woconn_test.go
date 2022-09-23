package won_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/workoak/wop/won"
)

var _ won.WOConnDriver = (*FakeWODriver)(nil)
var _ won.WOConn = (*FakeWOConn)(nil)

type FakeWOConn struct {
	closed bool
	subs   map[string]func(data []byte) ([]byte, error)
}

func (woconn *FakeWOConn) Ping() (bool, error) {
	if woconn.closed {
		return false, won.ErrConnectionClosed
	}
	return true, nil
}

func (woconn *FakeWOConn) Close() error {
	woconn.closed = true
	return nil
}

func (woconn *FakeWOConn) Request(subj string, data []byte, timeout time.Duration) ([]byte, error) {
	hfunc, ok := woconn.subs[subj]
	if !ok {
		return nil, nil
	}
	return hfunc(data)
}

func (woconn *FakeWOConn) Subscribe(subj string, hf func(data []byte) ([]byte, error)) error {
	woconn.subs[subj] = hf
	return nil
}

func (woconn *FakeWOConn) PingSubscribe(pingSubject string) {

}

type FakeWODriver struct {
}

func (wodriver *FakeWODriver) GetPrefix() string {
	return "wo-fakedriver"
}

func (wodriver *FakeWODriver) NewWOConn(wourl *url.URL) (won.WOConn, error) {
	subs := make(map[string]func(data []byte) ([]byte, error))
	return &FakeWOConn{closed: false,
		subs: subs}, nil
}

//
type TestSrv interface {
	runWOLivlinessProbe(cmd *won.WOLivlinessProbe) (*won.WOLivlinessProbeResult, error)
}

var _ TestSrv = (*TestSrvImp)(nil)

type TestSrvImp struct {
	woconn won.WOConn
}

func (srv *TestSrvImp) runWOLivlinessProbe(cmd *won.WOLivlinessProbe) (*won.WOLivlinessProbeResult, error) {
	return nil, nil
}

func (srv *TestSrvImp) StartSrv() error {

	return nil
}

func newTestSrv(woconn won.WOConn) (*TestSrvImp, error) {
	srv := &TestSrvImp{woconn: woconn}
	return srv, nil
}

func init() {
	won.RegisterWOConnDriver(&FakeWODriver{})
}
func TestWODriverRegister(t *testing.T) {
	stubPrefix := "wo-fakedriver"
	drivers := won.WOConnDrivers()
	assert.Containsf(t, drivers, stubPrefix, "stub connection should be registered")
}
func TestDial(t *testing.T) {
	woconn, err := won.Dial("wo-fakedriver://testdial")
	require.NoErrorf(t, err, "want no error when dialing, got %v", err)
	assert.NotNil(t, woconn, "want a valid woconn, got nil")
	ping, err := woconn.Ping()
	assert.True(t, ping, "want valid ping, got failed ping")

	err = woconn.Close()
	require.NoErrorf(t, err, "want no error, got %v", err)
	ping, err = woconn.Ping()
	assert.ErrorIs(t, err, won.ErrConnectionClosed, "wanted conn closed err")

}

func TestRequest(t *testing.T) {
	woconn, err := won.Dial("wo-fakedriver://testrequest")
	defer woconn.Close()
	require.NoErrorf(t, err, "want no error when dialing,got %v", err)
	req1func := func(data []byte) ([]byte, error) {
		resp := "hello " + string(data)
		return []byte(resp), nil
	}
	subj := "wo.test.request1"
	woconn.Subscribe(subj, req1func)
	resp, err := woconn.Request(subj, []byte("workoak"), time.Second)
	require.NoErrorf(t, err, "want no error with request,got %v", err)
	want := []byte("hello workoak")
	assert.Equal(t, want, resp, "wanted %q, got %q", want, resp)

}

func TestSrvConnect(t *testing.T) {
	woconn, err := won.Dial("wo-fakedriver://testrequest")
	defer woconn.Close()
	require.NoErrorf(t, err, "want no error when dialing,got %v", err)

	srv, err := newTestSrv(woconn)
	srv.StartSrv()

}

//Test Service connection to node

//Test Serivce Cmd Registration

//Test Client connection to node

//Test Client Cmd call and response

//Service publish events

//Client subscribe to events

//Client rewceive events

//
