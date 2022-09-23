package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"

	"github.com/workoak/wop"
)

func main() {
	wd, err := os.Getwd()
	specpath := filepath.Join(wd, "woncore.jsonc")
	specfile, err := os.Open(specpath)
	defer specfile.Close()
	if err != nil {
		log.Fatalf("error reading spec file %v,%e", specpath, err)
	}

	srvDef, err := wop.BuildSrvDefFromJSON(specfile)
	if err != nil {
		log.Fatalf("error building SrvDef %v,%e", specpath, err)
	}

	result, err := wop.BuildProtoBuffDefFromSrvDef(srvDef)

	if err != nil {
		log.Fatalf("error BuildProtoBuffDefFromJSON file %v,%e", specpath, err)
	}
	buff := &bytes.Buffer{}
	err = wop.GenerateProtobuf3FromFileDesc(result, buff)
	if err != nil {
		log.Fatalf("error GenerateProtobuf3FromFileDesc file %v,%e", specpath, err)
	}
	proto := filepath.Join(wd, "..", "woncore.proto")
	err = os.WriteFile(proto, buff.Bytes(), os.ModePerm)
	if err != nil {
		log.Fatalf("error writing .proto file %v,%e", specpath, err)
	}

	pb3GOAPI, err := wop.GenerateGOProtoBuffAPIFromProto(buff.Bytes())
	if err != nil {
		log.Fatalf("got error[%v], wanted error[%v], error[%v]", err != nil, wd, err)
		return
	}

	if err == nil && len(pb3GOAPI) == 0 {
		log.Fatalf(" no GO Pb3 API generated ")
		return
	}

	goapi := filepath.Join(wd, "..", "woncore.pb.go")
	err = os.WriteFile(goapi, pb3GOAPI, os.ModePerm)
	if err != nil {
		log.Fatalf("error writing go pb3 api file %v,%e", specpath, err)
	}
}
