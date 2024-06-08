package wogcli

import (
	"bytes"
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/workoak/wop/wog"
	"github.com/workoak/wop/wou/validate"
)

func RunInitCmd(ctx context.Context, cmd *InitializeCmd) (*InitializeCmdResult, error) {
	bucket := validate.NewBucket()
	bucket.Validate("name", &cmd.Name, validate.NotBlank)
	bucket.Validate("namespace", &cmd.Namespace, validate.NotBlank)

	if !bucket.IsValid() {
		return nil, bucket.Error()

	}
	return &InitializeCmdResult{}, nil
}

func RunGenerateCmd(ctx context.Context, cmd *GenerateCmd) (*GenerateCmdResult, error) {
	bucket := validate.NewBucket()
	bucket.Validate("in", cmd.Input, validate.NotBlank)
	bucket.Validate("lang", cmd.Lang, validate.NotBlank, validate.OneOfRule("go", "js"))
	bucket.Validate("outdir", cmd.Outdir, validate.NotBlank)

	if !bucket.IsValid() {
		return nil, bucket.Error()
	}
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	var specpath string
	if filepath.IsAbs(cmd.Input) {
		specpath = cmd.Input
	} else {
		specpath = filepath.Join(wd, cmd.Input)
	}
	specfile, err := os.Open(specpath)
	if err != nil {
		return nil, err
	}
	defer specfile.Close()

	srvDef, err := wog.BuildSrvDefFromJSON(specfile)
	if err != nil {
		return nil, err
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
	//write protofile
	outdir := cmd.Outdir
	if !filepath.IsAbs(outdir) {
		outdir = filepath.Join(wd, outdir)
	}
	if _, err := os.Stat(outdir); os.IsNotExist(err) {
		err = os.MkdirAll(outdir, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}
	proto := filepath.Join(outdir, srvDef.Name+".proto")
	err = os.WriteFile(proto, buff.Bytes(), os.ModePerm)
	if err != nil {
		log.Fatalf("error writing .proto file %v,%e", specpath, err)
	}

	pb3GOAPI, err := wog.GenerateGOProtoBuffAPIFromProto(buff.Bytes())
	if err != nil {
		log.Fatalf("got error[%v], wanted error[%v], error[%v]", err != nil, wd, err)
	}

	if err == nil && len(pb3GOAPI) == 0 {
		log.Fatalf(" no GO Pb3 API generated ")
	}

	goapi := filepath.Join(outdir, srvDef.Name+".pb.go")
	err = os.WriteFile(goapi, pb3GOAPI, os.ModePerm)
	if err != nil {
		log.Fatalf("error writing go pb3 api file %v,%e", specpath, err)
	}

	return &GenerateCmdResult{}, nil
}
