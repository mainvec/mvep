package won_test

import (
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/mainvec/mvep/won"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		return nil, errors.New("subject not found")
	}
	return hfunc(data)
}

func (woconn *FakeWOConn) Subscribe(subj string, hf func(data []byte) ([]byte, error)) error {
	//In Fake conn, its really only one sub
	woconn.subs[subj] = hf
	return nil
}

func (woconn *FakeWOConn) PingSubscribe(pingSubject string) {

}

func (woconn *FakeWOConn) QueueSubscribe(subject string, queue string, hf func(data []byte) ([]byte, error)) error {
	//In Fake conn, no multiple subscriptions. Subscribe and QueueSubscribe are the same
	woconn.subs[subject] = hf
	return nil
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

type TestSrv interface {
	won.WOSrv
}

var _ TestSrv = (*TestSrvImp)(nil)

type TestSrvImp struct {
	woconn won.WOConn
}

type TestCLT interface {
	won.WOClt
}

var _ TestCLT = (*TestCLTImp)(nil)

type TestCLTImp struct {
	woswitch *won.WOSwitch
}

func (srv *TestSrvImp) RunWOLivlinessProbe(cmd *won.WOLivlinessProbe) (result *won.WOLivlinessProbeResult, err error) {

	result = &won.WOLivlinessProbeResult{}
	return result, err
}

func (srv *TestSrvImp) Initialize() error {
	//initialize
	//update service state
	//setuo shutdown/quit
	return nil
}

func (srv *TestSrvImp) Namespace() string {
	return "wotest"
}

func (srv *TestSrvImp) Name() string {
	return "wotestsrv"
}

func (srv *TestSrvImp) Serve(woconn won.WOConn) error {
	srv.woconn = woconn
	//register commands/events for this Srv
	err := won.PrepareServe(srv, woconn)
	//Register commons ones first
	if err != nil {
		return err
	}
	//srv.woconn.Subscribe("wo.")
	return nil
}

func (clt *TestCLTImp) Namespace() string {
	return "wotest"
}

func (clt *TestCLTImp) Name() string {
	return "wotestsrv"
}

func (clt *TestCLTImp) RunWOLivlinessProbe(cmd *won.WOLivlinessProbe) (*won.WOLivlinessProbeResult, error) {
	result := &won.WOLivlinessProbeResult{}
	err := clt.woswitch.SendWOCmd(won.GetCmdSubject(clt, "WOLivlinessProbe"), cmd, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (clt *TestCLTImp) PingSRV() (string, error) {
	result, err := clt.woswitch.GetWOConn().Request(won.GetWOSrvPingSubject(clt), []byte("PING"), time.Millisecond*10)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

/*

func (wodb2srv *WODB2SRVImp) registerCmds() {
	//Register Ping
	wodb2srv.woconn.PingSubscribe("wo.wodb2srv.PING")
	wodb2srv.woconn.QueueSubscribe("wo.wodb2srv.cmd.WODbDqlCmd", "WODbDqlCmd", &wodb2api.WODbDqlCmd{}, wodb2srv.processCmd)
	wodb2srv.woconn.QueueSubscribe("wo.wodb2srv.cmd.WODbDmlCmd", "WODbDmlCmd", &wodb2api.WODbDmlCmd{}, wodb2srv.processCmd)
}
*/

func newTestSrv(woconn won.WOConn) (*TestSrvImp, error) {
	srv := &TestSrvImp{woconn: woconn}
	return srv, nil
}
func newTestClient(woconn won.WOConn) (*TestCLTImp, error) {
	woswitch, err := won.ConnectNewSwitch(woconn)
	if err != nil {
		return nil, err
	}
	clt := &TestCLTImp{woswitch: woswitch}
	return clt, nil
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
	resp, err := woconn.Request(subj, []byte("mainvec"), time.Second)
	require.NoErrorf(t, err, "want no error with request,got %v", err)
	want := []byte("hello mainvec")
	assert.Equal(t, want, resp, "wanted %q, got %q", want, resp)

}

func TestSrvConnect(t *testing.T) {
	woconn, err := won.Dial("wo-fakedriver://testrequest")
	require.NoError(t, err, "want no error")
	defer woconn.Close()
	require.NoErrorf(t, err, "want no error when dialing,got %v", err)

	srv, err := newTestSrv(woconn)
	require.NoError(t, err, "want no error")
	err = srv.Initialize()
	require.NoError(t, err, "want no error in starting SRV")
	//connCtx,  add it later
	err = srv.Serve(woconn)
	require.NoError(t, err, "want no error in starting SRV")

	//Do a client now
	clt, err := newTestClient(woconn)
	require.NoError(t, err, "want no error in starting SRV")
	pong, err := clt.PingSRV()
	require.NoError(t, err, "want no error")
	assert.NotEmpty(t, pong, "want non empty ping result")

	cmd := &won.WOLivlinessProbe{}

	cmdResult, err := clt.RunWOLivlinessProbe(cmd)
	require.NoError(t, err, "want no error with RunWOLivlinessProbe")
	assert.NotEmpty(t, cmdResult, "want non emoty cmd result")

	// // How about seperate Srv.start from Srv.Serve(connect)
	// http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	// })

	// http.HandleFunc("/hi", func(w http.ResponseWriter, r *http.Request) {
	// 	fmt.Fprintf(w, "Hi")
	// })

	// log.Fatal(http.ListenAndServe(":8081", nil))

}

//Test Service connection to node

//Test Serivce Cmd Registration

//Test Client connection to node

//Test Client Cmd call and response

//Service publish events

//Client subscribe to events

//Client rewceive events

//
