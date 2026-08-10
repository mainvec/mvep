package toolkit

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

	"github.com/mainvec/ugo/validate"
)

// dirPerm and filePerm are the permissions for generated output. Files are not
// world-writable (os.ModePerm/0777 was a defect).
const (
	dirPerm  = 0o755
	filePerm = 0o644
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
		return fmt.Errorf("get working directory: %w", err)
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
		err = os.MkdirAll(outdir, dirPerm)
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
	err := os.MkdirAll(jsApiDir, dirPerm)
	if err != nil {
		return fmt.Errorf("creating js api dir %s: %w", jsApiDir, err)
	}

	// Generate JavaScript classes
	jsClasses, err := GenerateJSVanillaClasses(srvDef)
	if err != nil {
		return fmt.Errorf("generating JS classes: %w", err)
	}
	if len(jsClasses) == 0 {
		return errors.New("no JS classes generated")
	}
	jsClassesFile := filepath.Join(jsApiDir, srvDef.Name+".js")
	err = os.WriteFile(jsClassesFile, jsClasses, filePerm)
	if err != nil {
		return fmt.Errorf("writing JS classes file %s: %w", jsClassesFile, err)
	}

	// Generate TypeScript definitions
	tsTypes, err := GenerateTSDefinitions(srvDef)
	if err != nil {
		return fmt.Errorf("generating TS definitions: %w", err)
	}
	if len(tsTypes) == 0 {
		return errors.New("no TS definitions generated")
	}
	tsTypesFile := filepath.Join(jsApiDir, srvDef.Name+".d.ts")
	err = os.WriteFile(tsTypesFile, tsTypes, filePerm)
	if err != nil {
		return fmt.Errorf("writing TS definitions file %s: %w", tsTypesFile, err)
	}

	// Generate package utilities
	jsPkgFile := filepath.Join(jsApiDir, srvDef.Name+"_package.js")
	if allow, _ := isAllowMVGen(jsPkgFile); allow {
		jsPkg, err := GenerateJSPackage(srvDef)
		if err != nil {
			return fmt.Errorf("generating JS package: %w", err)
		}
		err = os.WriteFile(jsPkgFile, jsPkg, filePerm)
		if err != nil {
			return fmt.Errorf("writing JS package file %s: %w", jsPkgFile, err)
		}
	}

	return nil
}

func executeGenerateGo(srvDef *SrvDef, outdir string, specpath string, format string, skipCmd bool) error {
	// T5: fail fast on constructs the runtime descriptor cannot represent,
	// before any template executes. The error names the offending command (or
	// record) and field, rather than panicking deep in template execution.
	if err := validateDescriptorRepresentable(srvDef); err != nil {
		return err
	}

	goApiDir := filepath.Join(outdir, "api")
	err := os.MkdirAll(goApiDir, dirPerm)
	if err != nil {
		return fmt.Errorf("creating go api dir %s: %w", goApiDir, err)
	}

	// Determine format from flag or gen_options
	formatOpt, hasFormatOpt := srvDef.GenOpts["format"]
	if format == "" && hasFormatOpt {
		format = formatOpt
	}
	// Require format to be specified
	if format == "" {
		return errors.New("format parameter is required (supported: plain, pb3)")
	}

	if format == "plain" {
		// Plain mode: generate plain Go structs with JSON tags (no protobuf)
		plainAPI, err := GenerateGOVanillaStructs(srvDef)
		if err != nil {
			return fmt.Errorf("generating plain Go structs: %w", err)
		}
		if len(plainAPI) == 0 {
			return errors.New("no plain Go structs generated")
		}
		plainFile := filepath.Join(goApiDir, srvDef.Name+".plain.go")
		err = os.WriteFile(plainFile, formatGoSource(plainFile, plainAPI), filePerm)
		if err != nil {
			return fmt.Errorf("writing plain Go file %s: %w", plainFile, err)
		}
	} else {
		// Protobuf mode: generate .proto and .pb.go files
		result, err := BuildProtoBuffDefFromSrvDef(srvDef)
		if err != nil {
			return fmt.Errorf("building protobuf definition from %s: %w", specpath, err)
		}
		buff := &bytes.Buffer{}
		err = GenerateProtobuf3FromFileDesc(result, buff)
		if err != nil {
			return fmt.Errorf("generating protobuf3 from file desc %s: %w", specpath, err)
		}

		proto := filepath.Join(goApiDir, srvDef.Name+".proto")
		err = os.WriteFile(proto, buff.Bytes(), filePerm)
		if err != nil {
			return fmt.Errorf("writing .proto file %s: %w", specpath, err)
		}

		pb3GOAPI, err := GenerateGOProtoBuffAPIFromProto(srvDef, buff.Bytes())
		if err != nil {
			return fmt.Errorf("generating GO protobuf API: %w", err)
		}

		if len(pb3GOAPI) == 0 {
			return errors.New("no GO pb3 API generated")
		}

		gopb3api := filepath.Join(goApiDir, srvDef.Name+".pb.go")
		err = os.WriteFile(gopb3api, pb3GOAPI, filePerm)
		if err != nil {
			return fmt.Errorf("writing go pb3 api file %s: %w", specpath, err)
		}
	}

	if !skipCmd {
		goMainCmdDir := filepath.Join(outdir, "cmd", srvDef.Name)
		err = os.MkdirAll(goMainCmdDir, dirPerm)
		if err != nil {
			return fmt.Errorf("creating go main cmd dir %s: %w", goMainCmdDir, err)
		}
		// cli main is protectable: honor NOMVEP/NOMVGEN/NOWOGEN markers so
		// hand-customized entry points are not overwritten on regeneration.
		gocliMainFile := filepath.Join(goMainCmdDir, srvDef.Name+"_main_cmd.go")
		if allow, _ := isAllowMVGen(gocliMainFile); allow {
			gocli, err := GenerateFromEmbeddTemplate(srvDef, "go_cli_main", "resources/codegen_templates/go/go_cli_main.txt")
			if err != nil {
				return fmt.Errorf("generating go cli: %w", err)
			}
			err = os.WriteFile(gocliMainFile, formatGoSource(gocliMainFile, gocli), filePerm)
			if err != nil {
				return fmt.Errorf("writing go cli file %s: %w", gocliMainFile, err)
			}
		}

		// generate version file (generate-once, protected by isAllowMVGen)
		goVersionFile := filepath.Join(goMainCmdDir, srvDef.Name+"_version.go")
		if allow, _ := isAllowMVGen(goVersionFile); allow {
			goVersion, err := GenerateFromEmbeddTemplate(srvDef, "go_cli_version", "resources/codegen_templates/go/go_cli_version.txt")
			if err != nil {
				return fmt.Errorf("generating go cli version: %w", err)
			}
			err = os.WriteFile(goVersionFile, formatGoSource(goVersionFile, goVersion), filePerm)
			if err != nil {
				return fmt.Errorf("writing go cli version file %s: %w", goVersionFile, err)
			}
		}
	}

	//package
	goPkgFile := filepath.Join(goApiDir, srvDef.Name+"_package.go")

	if allow, _ := isAllowMVGen(goPkgFile); allow {
		gopkg, err := GenerateFromEmbeddTemplate(srvDef, "go_pkg", "resources/codegen_templates/go/go_package_code.txt")
		if err != nil {
			return fmt.Errorf("generating go package: %w", err)
		}

		err = os.WriteFile(goPkgFile, formatGoSource(goPkgFile, gopkg), filePerm)
		if err != nil {
			return fmt.Errorf("writing go package file %s: %w", goPkgFile, err)
		}

	}

	//go default implemetaiton
	goImplFile := filepath.Join(outdir, srvDef.Name+"_impl.go")

	if allow, _ := isAllowMVGen(goImplFile); allow {
		goimpl, err := GenerateFromEmbeddTemplate(srvDef, "go_impl", "resources/codegen_templates/go/go_impl_code.txt")
		if err != nil {
			return fmt.Errorf("generating go impl: %w", err)
		}

		err = os.WriteFile(goImplFile, formatGoSource(goImplFile, goimpl), filePerm)
		if err != nil {
			return fmt.Errorf("writing go impl file %s: %w", goImplFile, err)
		}
	}

	//generate package commands runner
	if err := generateFileFromTemplate(srvDef, "go_commands_runner", "resources/codegen_templates/go/go_commands_runner_code.txt", outdir, srvDef.Name+"_commands.go"); err != nil {
		return err
	}

	return nil
}

func generateFileFromTemplate(srvDef *SrvDef, tmplName string, tmplPath string, outdir string, filename string) error {
	//go default implemetaiton
	outFile := filepath.Join(outdir, filename)

	if allow, _ := isAllowMVGen(outFile); allow {
		fileContent, err := GenerateFromEmbeddTemplate(srvDef, tmplName, tmplPath)
		if err != nil {
			return fmt.Errorf("generating %s: %w", tmplName, err)
		}

		if strings.HasSuffix(filename, ".go") {
			fileContent = formatGoSource(outFile, fileContent)
		}
		err = os.WriteFile(outFile, fileContent, filePerm)
		if err != nil {
			return fmt.Errorf("writing %s to file %s: %w", tmplName, outFile, err)
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
			log.Printf("could not check file[%v] for toolkit, err:%v", fileName, err)
			return false, err
		}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		//read only first line
		//look for NOMVEP marker in first line (NOMVGEN/NOWOGEN kept for legacy support)
		line := strings.ToUpper(scanner.Text())
		if strings.Contains(line, "NOMVEP") || strings.Contains(line, "NOMVGEN") || strings.Contains(line, "NOWOGEN") {
			log.Printf("skipping generation for file [%v]. NOMVEP (or legacy NOMVGEN/NOWOGEN) found in first line", fileName)
			return false, nil
		}
		break
	}

	if err := scanner.Err(); err != nil {
		log.Printf("could not check first line in file[%v] for toolkit, err:%v", fileName, err)
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
		return nil, fmt.Errorf("get working directory: %w", err)
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
