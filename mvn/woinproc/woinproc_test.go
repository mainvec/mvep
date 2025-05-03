package woinproc_test

import (
	"log"
	"net/url"
	"testing"

	"github.com/mainvec/mvep/won"
	"github.com/mainvec/mvep/won/woinproc"
	"github.com/stretchr/testify/require"
)

func TestNewConn(t *testing.T) {
	driver := &woinproc.WOInProcConnDriver{}
	conn_url := "wo-inproc://test1"
	u, err := url.Parse(conn_url)
	if err != nil {
		log.Fatalf("unexpected url parse")
		return
	}
	scheme := u.Scheme
	if scheme != driver.GetPrefix() {
		t.Errorf("got schema[%v], wanted[%v]", driver.GetPrefix(), scheme)
	}
	woconn, err := driver.NewWOConn(u)
	if err != nil {
		t.Errorf("unexpected error for NewWOConn,%v", err)
		return
	}
	pong, err := woconn.Ping()
	require.NoError(t, err, "error during ping")
	require.True(t, pong, "got ping[%v], wanted[%v]", pong, false)

	err = woconn.Close()
	require.NoError(t, err, "error during closing")
	pong, err = woconn.Ping()
	require.Error(t, err, "connection is closed, should return an error ")
	//call again,
}

func TestQueueSubscribe(t *testing.T) {
	woconn, err := getTestInProcConn()
	if err != nil {
		t.Fatalf("error getting test woconn,%v", err)
	}
	defer woconn.Close()

}

func getTestInProcConn() (won.WOConn, error) {
	driver := &woinproc.WOInProcConnDriver{}
	conn_url := "wo-inproc://test1"
	u, err := url.Parse(conn_url)
	if err != nil {
		log.Fatalf("unexpected url parse")
	}
	woconn, err := driver.NewWOConn(u)
	if err != nil {
		return nil, err
	}
	return woconn, nil

}
