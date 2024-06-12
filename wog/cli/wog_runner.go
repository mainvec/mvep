package wogcli

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/workoak/wop/wog"
	"github.com/workoak/woutil/validate"
)

func RunInitializeCmd(ctx context.Context, cmd *InitializeCmd) (*InitializeCmdResult, error) {
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
	bucket.Validate("in", cmd.In, validate.NotBlank)
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
	if filepath.IsAbs(cmd.In) {
		specpath = cmd.In
	} else {
		specpath = filepath.Join(wd, cmd.In)
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

	gopb3api := filepath.Join(outdir, srvDef.Name+".pb.go")
	err = os.WriteFile(gopb3api, pb3GOAPI, os.ModePerm)
	if err != nil {
		log.Fatalf("error writing go pb3 api file %v,%e", specpath, err)
	}

	goapi, err := wog.GenerateGOAPI(srvDef)
	if err != nil {
		log.Fatalf("error generating go api: %v", err)
	}
	goapiFile := filepath.Join(outdir, srvDef.Name+"_api.go")
	err = os.WriteFile(goapiFile, goapi, os.ModePerm)
	if err != nil {
		log.Fatalf("error writing go api file %v,%e", goapiFile, err)
	}

	gocli, err := wog.GenerateFromEmbeddTemplate(srvDef, "go_cli_main", "resources/codegen_templates/go/go_cli_main.txt")
	if err != nil {
		log.Fatalf("error generating go cli: %v", err)
	}
	goMainCmdDir := filepath.Join(outdir, "cmd")
	err = os.MkdirAll(goMainCmdDir, os.ModePerm)
	if err != nil {
		log.Fatalf("error creating go maoin cmd dir: %v,%v", goMainCmdDir, err)
	}
	gocliMainFile := filepath.Join(goMainCmdDir, srvDef.Name+"_main_cmd.go")
	err = os.WriteFile(gocliMainFile, gocli, os.ModePerm)
	if err != nil {
		log.Fatalf("error writing go cli file %v,%e", gocliMainFile, err)
	}

	return &GenerateCmdResult{}, nil
}

func RunValidateCmd(ctx context.Context, cmd *ValidateCmd) (*ValidateCmdResult, error) {
	bucket := validate.NewBucket()
	bucket.Validate("in", cmd.In, validate.NotBlank)
	if !bucket.IsValid() {
		return nil, bucket.Error()
	}
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	var specpath string
	if filepath.IsAbs(cmd.In) {
		specpath = cmd.In
	} else {
		specpath = filepath.Join(wd, cmd.In)
	}
	specfile, err := os.Open(specpath)
	if err != nil {
		return nil, err
	}
	defer specfile.Close()
	res, err := wog.ValidateJSONSchema(specfile)
	if err != nil {
		return nil, err
	}
	cmdResult := &ValidateCmdResult{}
	cmdResult.Valid = res.Valid()
	if !res.Valid() {
		errs := make([]string, len(res.ValidationErrors()))
		for _, e := range res.ValidationErrors() {
			errs = append(errs, e.String())
		}
		cmdResult.Errors = errs
		fmt.Fprintf(os.Stderr, "WOP Spec is InValid: %v\n", errs)
	} else {
		fmt.Printf("valid!.\n")
	}
	return cmdResult, err
}
