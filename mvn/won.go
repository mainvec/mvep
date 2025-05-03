package won

import (
	"errors"
	"strings"

	pb "github.com/mainvec/mvep/won/protobuf"
)

const WOSYS_NODE_PING_SUB = "$wosys.wonode.ping"

var (
	ErrConnectionClosed = errors.New("connection is already closed")
)

type WOSrvInfo interface {
	Namespace() string
	Name() string
}

type WOSrv interface {
	WOSrvInfo
	Initialize() error
	RunWOLivlinessProbe(cmd *WOLivlinessProbe) (*WOLivlinessProbeResult, error)
}

type WOClt interface {
	WOSrvInfo
	PingSRV() (string, error)
	RunWOLivlinessProbe(cmd *WOLivlinessProbe) (*WOLivlinessProbeResult, error)
}

func GetWOSrvPingSubject(srvInfo WOSrvInfo) string {
	return strings.Join([]string{srvInfo.Namespace(), srvInfo.Name(), "PING"}, ".")
}

func PrepareServe(srv WOSrv, woconn WOConn) error {

	handlePing := func(data []byte) ([]byte, error) {

		return []byte("PONG"), nil
	}

	err := woconn.Subscribe(GetWOSrvPingSubject(srv), handlePing)
	if err != nil {
		return err
	}
	//"wo.wodb2srv.cmd.WODbDqlCmd", "WODbDqlCmd", &wodb2api.WODbDqlCmd{}, wodb2srv.processCmd
	//Every service shoulw have a factory, used to create instances. Not encoding specific.
	woLiveProbSubj := GetCmdSubject(srv, "WOLivlinessProbe")
	err = woconn.QueueSubscribe(woLiveProbSubj, woLiveProbSubj,
		buildWOReqHandler(woLiveProbSubj, func() any { return &WOLivlinessProbe{} }, processWOSrvCmd(srv)))

	return err
}

func GetCmdSubject(srvInfo WOSrvInfo, cmdName string) string {
	//Assert string is not blank
	return BuildSubject(srvInfo.Namespace(), srvInfo.Name(), "cmd", cmdName)
}

func BuildSubject(subjItems ...string) string {
	return strings.Join(subjItems, ".")
}

func buildWOReqHandler(subject string, woCmdCreator func() any, wocmdProcess func(wocmd any) (any, error)) func(reqData []byte) (respData []byte, err error) {
	//Data is a WOReq
	//Response should be WOResp

	woreqHandler := func(reqData []byte) (respData []byte, err error) {
		// 1- decode req to WOReq
		woreq := &WOReq{}
		err = pb.Decode(subject, reqData, woreq)

		if err != nil {
			//may this should an error response and nor golang error
			return nil, err
		}

		// 2- decode WOReq.payload to CMD
		wocmd := woCmdCreator()
		//HMM.. check of correct woCMD type?
		err = pb.Decode(subject, woreq.Payload, wocmd)
		if err != nil {
			return nil, err
		}
		// 3- call the Cmd function
		woresult, err := wocmdProcess(wocmd)
		if err != nil {
			return nil, err
		}
		// 4- encode result
		wocmdresultPayload, err := pb.Encode(subject, woresult)
		if err != nil {

			return nil, err
		}
		// 5- build WOResp
		woResp := &WOResp{
			Payload: wocmdresultPayload,
		}

		// 6- encode WOResp and return
		respData, err = pb.Encode(subject, woResp)
		if err != nil {
			return nil, err
		}
		return respData, nil
	}
	return woreqHandler

}

func processWOSrvCmd(srv WOSrv) func(wocmd any) (any, error) {

	wocmdProc := func(wocmd any) (any, error) {
		switch v := wocmd.(type) {
		case *WOLivlinessProbe:
			return srv.RunWOLivlinessProbe(v)

		default:
			return nil, errors.New("wrong cmd")
		}
	}
	return wocmdProc
}
