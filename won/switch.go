package won

type WOSwitch struct {
	woconn WOConn
}

func NewSwitch(wourl string) (*WOSwitch, error) {
	woconn, err := Dial(wourl)

	if err != nil {
		return nil, err
	}
	return &WOSwitch{
		woconn: woconn,
	}, nil
}

func (woswitch *WOSwitch) Ping() (bool, error) {
	return woswitch.woconn.Ping()
}
