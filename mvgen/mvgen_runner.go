package mvgen

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mainvec/ugo/validate"
)

func ExeucuteInitializeCmd(ctx context.Context, cmdName string, cmdNamespace string) error {
	bucket := validate.NewBucket()
	bucket.Validate("name", &cmdName, validate.NotBlank)
	bucket.Validate("namespace", &cmdNamespace, validate.NotBlank)

	if !bucket.IsValid() {
		return bucket.Error()

	}
	return nil
}

func ExecuteGenerate(ctx context.Context, cmdIn string, cmdOutdir string, cmdLang string, skipCmd bool, format string) error {

	bucket := validate.NewBucket()
	bucket.Validate("in", cmdIn, validate.NotBlank)
	bucket.Validate("lang", cmdLang, validate.NotBlank)
	bucket.Validate("outdir", cmdOutdir, validate.NotBlank)

	if !bucket.IsValid() {
		return bucket.Error()
	}

	// Parse comma-separated languages and validate each
	langs := strings.Split(cmdLang, ",")
	for i, lang := range langs {
		langs[i] = strings.TrimSpace(lang)
		if langs[i] != "go" && langs[i] != "js" {
			return errors.New("invalid lang value: " + langs[i] + " (supported: go, js)")
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	var specpath string
	if filepath.IsAbs(cmdIn) {
		specpath = cmdIn
	} else {
		specpath = filepath.Join(wd, cmdIn)
	}
	specfile, err := os.Open(specpath)
	if err != nil {
		return err
	}
	defer specfile.Close()

	srvDef, err := BuildSrvDefFromJSON(specfile)
	if err != nil {
		return err
	}

	//write protofile
	outdir := cmdOutdir
	if !filepath.IsAbs(outdir) {
		outdir = filepath.Join(wd, outdir)
	}
	if _, err := os.Stat(outdir); os.IsNotExist(err) {
		err = os.MkdirAll(outdir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	// Generate for each specified language
	for _, lang := range langs {
		switch lang {
		case "js":
			if err := executeGenerateJS(srvDef, outdir, format); err != nil {
				return err
			}
		case "go":
			if err := executeGenerateGo(srvDef, outdir, specpath, format, skipCmd); err != nil {
				return err
			}
		}
	}

	return nil
}

func executeGenerateJS(srvDef *SrvDef, outdir string, format string) error {
	jsApiDir := filepath.Join(outdir, "api")
	err := os.MkdirAll(jsApiDir, os.ModePerm)
	if err != nil {
		log.Fatalf("error creating js api dir: %v,%v", jsApiDir, err)
	}

	// Generate JavaScript classes
	jsClasses, err := GenerateJSVanillaClasses(srvDef)
	if err != nil {
		log.Fatalf("error generating JS classes: %v", err)
	}
	if len(jsClasses) == 0 {
		log.Fatalf("no JS classes generated")
	}
	jsClassesFile := filepath.Join(jsApiDir, srvDef.Name+".js")
	err = os.WriteFile(jsClassesFile, jsClasses, os.ModePerm)
	if err != nil {
		log.Fatalf("error writing JS classes file %v,%e", jsClassesFile, err)
	}

	// Generate TypeScript definitions
	tsTypes, err := GenerateTSDefinitions(srvDef)
	if err != nil {
		log.Fatalf("error generating TS definitions: %v", err)
	}
	if len(tsTypes) == 0 {
		log.Fatalf("no TS definitions generated")
	}
	tsTypesFile := filepath.Join(jsApiDir, srvDef.Name+".d.ts")
	err = os.WriteFile(tsTypesFile, tsTypes, os.ModePerm)
	if err != nil {
		log.Fatalf("error writing TS definitions file %v,%e", tsTypesFile, err)
	}

	// Generate package utilities
	jsPkgFile := filepath.Join(jsApiDir, srvDef.Name+"_package.js")
	if allow, _ := isAllowMVGen(jsPkgFile); allow {
		jsPkg, err := GenerateJSPackage(srvDef)
		if err != nil {
			log.Fatalf("error generating JS package: %v", err)
		}
		err = os.WriteFile(jsPkgFile, jsPkg, os.ModePerm)
		if err != nil {
			log.Fatalf("error writing JS package file %v,%e", jsPkgFile, err)
		}
	}

	return nil
}

func executeGenerateGo(srvDef *SrvDef, outdir string, specpath string, format string, skipCmd bool) error {
	goApiDir := filepath.Join(outdir, "api")
	err := os.MkdirAll(goApiDir, os.ModePerm)
	if err != nil {
		log.Fatalf("error creating go api dir: %v,%v", goApiDir, err)
	}

	// Determine format from flag or gen_options
	formatOpt, hasFormatOpt := srvDef.GenOpts["format"]
	if format == "" && hasFormatOpt {
		format = formatOpt
	}
	// Require format to be specified
	if format == "" {
		log.Fatalf("format parameter is required (supported: plain, pb3)")
	}

	if format == "plain" {
		// Plain mode: generate plain Go structs with JSON tags (no protobuf)
		plainAPI, err := GenerateGOVanillaStructs(srvDef)
		if err != nil {
			log.Fatalf("error generating plain Go structs: %v", err)
		}
		if len(plainAPI) == 0 {
			log.Fatalf("no plain Go structs generated")
		}
		plainFile := filepath.Join(goApiDir, srvDef.Name+".plain.go")
		err = os.WriteFile(plainFile, plainAPI, os.ModePerm)
		if err != nil {
			log.Fatalf("error writing plain Go file %v,%e", plainFile, err)
		}
	} else {
		// Protobuf mode: generate .proto and .pb.go files
		result, err := BuildProtoBuffDefFromSrvDef(srvDef)
		if err != nil {
			log.Fatalf("error BuildProtoBuffDefFromJSON file %v,%e", specpath, err)
		}
		buff := &bytes.Buffer{}
		err = GenerateProtobuf3FromFileDesc(result, buff)
		if err != nil {
			log.Fatalf("error GenerateProtobuf3FromFileDesc file %v,%e", specpath, err)
		}

		proto := filepath.Join(goApiDir, srvDef.Name+".proto")
		err = os.WriteFile(proto, buff.Bytes(), os.ModePerm)
		if err != nil {
			log.Fatalf("error writing .proto file %v,%e", specpath, err)
		}

		pb3GOAPI, err := GenerateGOProtoBuffAPIFromProto(srvDef, buff.Bytes())
		if err != nil {
			log.Fatalf("error generating GO protobuf API: %v", err)
		}

		if err == nil && len(pb3GOAPI) == 0 {
			log.Fatalf(" no GO Pb3 API generated ")
		}

		gopb3api := filepath.Join(goApiDir, srvDef.Name+".pb.go")
		err = os.WriteFile(gopb3api, pb3GOAPI, os.ModePerm)
		if err != nil {
			log.Fatalf("error writing go pb3 api file %v,%e", specpath, err)
		}
	}

	// goapi, err := GenerateGOAPI(srvDef)
	// if err != nil {
	// 	log.Fatalf("error generating go api: %v", err)
	// }
	// goapiFile := filepath.Join(outdir, srvDef.Name+"_api.go")
	// err = os.WriteFile(goapiFile, goapi, os.ModePerm)
	// if err != nil {
	// 	log.Fatalf("error writing go api file %v,%e", goapiFile, err)
	// }

	if !skipCmd {
		gocli, err := GenerateFromEmbeddTemplate(srvDef, "go_cli_main", "resources/codegen_templates/go/go_cli_main.txt")
		if err != nil {
			log.Fatalf("error generating go cli: %v", err)
		}
		goMainCmdDir := filepath.Join(outdir, "cmd", srvDef.Name)
		err = os.MkdirAll(goMainCmdDir, os.ModePerm)
		if err != nil {
			log.Fatalf("error creating go main cmd dir: %v,%v", goMainCmdDir, err)
		}
		gocliMainFile := filepath.Join(goMainCmdDir, srvDef.Name+"_main_cmd.go")
		err = os.WriteFile(gocliMainFile, gocli, os.ModePerm)
		if err != nil {
			log.Fatalf("error writing go cli file %v,%e", gocliMainFile, err)
		}
	}

	//package
	goPkgFile := filepath.Join(goApiDir, srvDef.Name+"_package.go")

	if allow, _ := isAllowMVGen(goPkgFile); allow {
		gopkg, err := GenerateFromEmbeddTemplate(srvDef, "go_pkg", "resources/codegen_templates/go/go_package_code.txt")
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

	if allow, _ := isAllowMVGen(goImplFile); allow {
		goimpl, err := GenerateFromEmbeddTemplate(srvDef, "go_impl", "resources/codegen_templates/go/go_impl_code.txt")
		if err != nil {
			log.Fatalf("error generating go impl: %v", err)
		}

		err = os.WriteFile(goImplFile, goimpl, os.ModePerm)
		if err != nil {
			log.Fatalf("error writing go impl file %v,%e", goImplFile, err)
		}
	}

	//generate package commands runner
	generateFileFromTemplate(srvDef, "go_commands_runner", "resources/codegen_templates/go/go_commands_runner_code.txt", outdir, srvDef.Name+"_commands.go")

	return nil
}

func generateFileFromTemplate(srvDef *SrvDef, tmplName string, tmplPath string, outdir string, filename string) error {
	//go default implemetaiton
	outFile := filepath.Join(outdir, filename)

	if allow, _ := isAllowMVGen(outFile); allow {
		fileContent, err := GenerateFromEmbeddTemplate(srvDef, tmplName, tmplPath)
		if err != nil {
			log.Fatalf("error generating [%v]: %v", tmplName, err)
		}

		err = os.WriteFile(outFile, fileContent, os.ModePerm)
		if err != nil {
			log.Fatalf("error writing [%v] to file %v:%e", tmplName, outFile, err)
		}
	}
	return nil
}

func isAllowMVGen(fileName string) (bool, error) {
	if len(fileName) == 0 {
		return false, errors.New("invalid filename")
	}
	file, err := os.Open(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			//file doesn ot exists, allow generation
			return true, nil
		} else {
			log.Printf("could not check file[%v] for mvgen, err:%v", fileName, err)
			return false, err
		}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		//read only first line
		//look for  NOMVGEN and NOWOGEN comment in first line
		//NOWOGEN is left for legacy support
		line := strings.ToUpper(scanner.Text())
		if strings.Contains(line, "NOMVGEN") || strings.Contains(line, "NOWOGEN") {
			log.Printf("skipping generation for file [%v]. NOMVGEN (or NOWOGEN) found in first line", fileName)
			return false, nil
		}
		break
	}

	if err := scanner.Err(); err != nil {
		log.Printf("could not check first line in file[%v] for mvgen, err:%v", fileName, err)
		return false, err
	}
	return true, nil
}

func ExecuteValidateCmd(ctx context.Context, cmdIn string) (ValidationResult, error) {
	bucket := validate.NewBucket()
	bucket.Validate("in", cmdIn, validate.NotBlank)
	if !bucket.IsValid() {
		return nil, bucket.Error()
	}
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	var specpath string
	if filepath.IsAbs(cmdIn) {
		specpath = cmdIn
	} else {
		specpath = filepath.Join(wd, cmdIn)
	}
	specfile, err := os.Open(specpath)
	if err != nil {
		return nil, err
	}
	defer specfile.Close()
	res, err := ValidateJSONSchema(specfile)
	if err != nil {
		return nil, err
	}
	// cmdResult := &api.ValidateCmdResult{}
	// cmdResult.Valid = res.Valid()
	// if !res.Valid() {
	// 	errs := make([]string, len(res.ValidationErrors()))
	// 	for _, e := range res.ValidationErrors() {
	// 		errs = append(errs, e.String())
	// 	}
	// 	cmdResult.Errors = errs
	// 	fmt.Fprintf(os.Stderr, "MVEP Spec is InValid: %v\n", errs)
	// } else {
	// 	fmt.Printf("valid!.\n")
	// }
	return res, nil
}
