package won_test

import (
	"testing"

	"github.com/workoak/wop/won"
	_ "github.com/workoak/wop/won/woinproc"
)

func TestSwitchNew(t *testing.T) {
	//a connect to a node, using node url

	woswitch, err := won.NewSwitch("wo-inproc://inproc-test")
	if err != nil {
		t.Errorf("error creating a new WO Switch, %v", err)
		return
	}
	if woswitch == nil {
		t.Error("no Switch retunred ")
		return
	}
	pong, err := woswitch.Ping()
	if err != nil {
		t.Errorf("error during Ping, %v", err)
		return
	}
	if !pong {
		t.Errorf("got pong[%v], wanted[%v]", pong, true)
		return
	}

}

func TestSubscribe(t *testing.T) {
	//get a connection
	//subscribe to a subject, with WOContext function
	//
}
