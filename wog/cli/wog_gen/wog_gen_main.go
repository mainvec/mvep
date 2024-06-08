package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"

	"github.com/workoak/wop/wog"
)

// WOG CLI self-generator. to be run manually when needed.
func main() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	specpath := filepath.Join(wd, "wogcli.jsonc")
	specfile, err := os.Open(specpath)
	if err != nil {
		log.Fatalf("error reading spec file %v,%e", specpath, err)
	}
	defer specfile.Close()

	srvDef, err := wog.BuildSrvDefFromJSON(specfile)
	if err != nil {
		log.Fatalf("error building SrvDef %v,%e", specpath, err)
	}

	result, err := wog.BuildProtoBuffDefFromSrvDef(srvDef)

	if err != nil {
		log.Fatalf("error BuildProtoBuffDefFromJSON file %v,%e", specpath, err)
	}
	buff := &bytes.Buffer{}
	err = wog.GenerateProtobuf3FromFileDesc(result, buff)
	if err != nil {
		log.Fatalf("error GenerateProtobuf3FromFileDesc file %v,%e", specpath, err)
	}
	proto := filepath.Join(wd, "..", "wogcli.proto")
	err = os.WriteFile(proto, buff.Bytes(), os.ModePerm)
	if err != nil {
		log.Fatalf("error writing .proto file %v,%e", specpath, err)
	}

	pb3GOAPI, err := wog.GenerateGOProtoBuffAPIFromProto(buff.Bytes())
	if err != nil {
		log.Fatalf("got error[%v], wanted error[%v], error[%v]", err != nil, wd, err)
		return
	}

	if err == nil && len(pb3GOAPI) == 0 {
		log.Fatalf(" no GO Pb3 API generated ")
		return
	}

	goapi := filepath.Join(wd, "..", "wog.pb.go")
	err = os.WriteFile(goapi, pb3GOAPI, os.ModePerm)
	if err != nil {
		log.Fatalf("error writing go pb3 api file %v,%e", specpath, err)
	}
}
