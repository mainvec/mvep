package wogcli

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/workoak/wogo/validate"
	"github.com/workoak/wop/wog"
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
	goApiDir := filepath.Join(outdir, "api")
	err = os.MkdirAll(goApiDir, os.ModePerm)
	if err != nil {
		log.Fatalf("error creating go api dir: %v,%v", goApiDir, err)
	}

	proto := filepath.Join(goApiDir, srvDef.Name+".proto")
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

	gopb3api := filepath.Join(goApiDir, srvDef.Name+".pb.go")
	err = os.WriteFile(gopb3api, pb3GOAPI, os.ModePerm)
	if err != nil {
		log.Fatalf("error writing go pb3 api file %v,%e", specpath, err)
	}

	// goapi, err := wog.GenerateGOAPI(srvDef)
	// if err != nil {
	// 	log.Fatalf("error generating go api: %v", err)
	// }
	// goapiFile := filepath.Join(outdir, srvDef.Name+"_api.go")
	// err = os.WriteFile(goapiFile, goapi, os.ModePerm)
	// if err != nil {
	// 	log.Fatalf("error writing go api file %v,%e", goapiFile, err)
	// }

	gocli, err := wog.GenerateFromEmbeddTemplate(srvDef, "go_cli_main", "resources/codegen_templates/go/go_cli_main.txt")
	if err != nil {
		log.Fatalf("error generating go cli: %v", err)
	}
	goMainCmdDir := filepath.Join(outdir, "cmd")
	err = os.MkdirAll(goMainCmdDir, os.ModePerm)
	if err != nil {
		log.Fatalf("error creating go main cmd dir: %v,%v", goMainCmdDir, err)
	}
	gocliMainFile := filepath.Join(goMainCmdDir, srvDef.Name+"_main_cmd.go")
	err = os.WriteFile(gocliMainFile, gocli, os.ModePerm)
	if err != nil {
		log.Fatalf("error writing go cli file %v,%e", gocliMainFile, err)
	}

	//package
	goPkgFile := filepath.Join(goApiDir, srvDef.Name+"_package.go")

	if allow, _ := isAllowWOGen(goPkgFile); allow {
		gopkg, err := wog.GenerateFromEmbeddTemplate(srvDef, "go_pkg", "resources/codegen_templates/go/go_package_code.txt")
		if err != nil {
			log.Fatalf("error generating go package: %v", err)
		}

		err = os.WriteFile(goPkgFile, gopkg, os.ModePerm)
		if err != nil {
			log.Fatalf("error writing go package file %v,%e", goPkgFile, err)
		}

	}

	//go default implemetaiton
	goImplFile := filepath.Join(outdir, srvDef.Name+"_impl.go")

	if allow, _ := isAllowWOGen(goImplFile); allow {
		goimpl, err := wog.GenerateFromEmbeddTemplate(srvDef, "go_impl", "resources/codegen_templates/go/go_impl_code.txt")
		if err != nil {
			log.Fatalf("error generating go impl: %v", err)
		}

		err = os.WriteFile(goImplFile, goimpl, os.ModePerm)
		if err != nil {
			log.Fatalf("error writing go impl file %v,%e", goImplFile, err)
		}

	}

	return &GenerateCmdResult{}, nil
}

func isAllowWOGen(fileName string) (bool, error) {
	if len(fileName) == 0 {
		return false, errors.New("invalid filename")
	}
	file, err := os.Open(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			//file doesn ot exists, allow generation
			return true, nil
		} else {
			log.Printf("could not check file[%v] for wogen, err:%v", fileName, err)
			return false, err
		}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		//read only first line
		//look for NOWOGEN comment in first line
		line := strings.ToUpper(scanner.Text())
		if strings.Contains(line, "NOWOGEN") {
			log.Printf("skipping generation for file [%v]. NOWOGEN found in first line", fileName)
			return false, nil
		}
		break
	}

	if err := scanner.Err(); err != nil {
		log.Printf("could not check first line in file[%v] for wogen, err:%v", fileName, err)
		return false, err
	}
	return true, nil
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
