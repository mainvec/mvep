package won

import (
	"time"

	pb "github.com/workoak/wop/won/protobuf"
)

type WOSwitch struct {
	woconn WOConn
}

func DialNewSwitch(wourl string) (*WOSwitch, error) {
	woconn, err := Dial(wourl)

	if err != nil {
		return nil, err
	}
	return ConnectNewSwitch(woconn)
}

func ConnectNewSwitch(woconn WOConn) (*WOSwitch, error) {

	return &WOSwitch{
		woconn: woconn,
	}, nil
}

func (woswitch *WOSwitch) Ping() (bool, error) {
	return woswitch.woconn.Ping()
}

func (woswitch *WOSwitch) GetWOConn() WOConn {
	return woswitch.woconn
}

func (woswitch *WOSwitch) SendWOCmd(subject string, v interface{}, returnValue interface{}) error {
	data, err := pb.Encode(subject, v)
	if err != nil {
		return err
	}
	req := &WOReq{Payload: data,
		Wocmdid: subject,
	}
	reqdata, err := pb.Encode(subject, req)
	if err != nil {
		return err
	}
	resp, err := woswitch.woconn.Request(subject, reqdata, 2*time.Second)
	if err != nil {
		return err
	}

	//response data is the WORes.
	//get the payload and decode for the returnedValue
	//Note: Feels inefficeint, there must be a better way!
	rdata := resp
	// errCheck := rdata[0:4]
	// if string(errCheck) == "ERR:" {
	// 	return errors.New(string(rdata))
	// }
	woresp := &WOResp{}

	err = pb.Decode(subject, rdata, woresp)

	if err != nil {
		return err
	}
	err = pb.Decode(subject, woresp.Payload, returnValue)
	if err != nil {
		return err
	}
	return nil
}
