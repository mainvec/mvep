package wog

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func GenerateGOProtoBuffAPIFromProto(protoContent []byte) ([]byte, error) {
	//Using protoc wiht

	protocPath, err := exec.LookPath("protoc")
	if err != nil {
		return nil, fmt.Errorf("error finding path for 'protoc' command: %v", err)
	}

	goapiDir, err := os.MkdirTemp("", "wop-go-temp-*")
	if err != nil {
		return nil, fmt.Errorf("error creating go temp dir: %v", err)
	}
	defer os.RemoveAll(goapiDir) // clean up
	woproto3File, err := os.CreateTemp(goapiDir, "wo-proto3-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("error creating go temp file: %v", err)
	}
	defer woproto3File.Close()
	woproto3File.Write(protoContent)
	//fpath := filepath.Join(goapiDir, woproto3File.Name())
	//,
	cmd := exec.Command(protocPath, "-I="+goapiDir, "--go_opt=paths=source_relative",
		"--go_out="+goapiDir, woproto3File.Name())
	cmd.Stdin = bytes.NewReader(protoContent)
	var out bytes.Buffer
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("error running protoc: %v,[%v]", err, out.String())
	}
	goAPIBytes, err := os.ReadFile(woproto3File.Name() + ".pb.go")
	if err != nil {
		return nil, fmt.Errorf("error reading go api temp file: %v", err)
	}
	return goAPIBytes, nil
}

func GenerateFromEmbeddTemplate(srvDef *SrvDef, templateName, templateEmbeddPath string) ([]byte, error) {

	templateReader, err := resources.Open(templateEmbeddPath)
	if err != nil {
		return nil, err
	}
	return GenerateFromTemplate(srvDef, templateName, templateReader)
}

func GenerateFromTemplate(srvDef *SrvDef, templateName string, templateReader io.Reader) ([]byte, error) {

	data := prepareTemplateDataMap(srvDef)

	tmpl, err := LoadTemplate(templateName, templateReader)
	if err != nil {

		return nil, err
	}
	srvBuff := &bytes.Buffer{}
	err = tmpl.Execute(srvBuff, data)
	if err != nil {
		return nil, err
	}
	return srvBuff.Bytes(), nil
}

func GenerateGOSRV(srvDef *SrvDef) ([]byte, error) {
	srvname := srvDef.Name + "srv"
	return GenerateFromEmbeddTemplate(srvDef, srvname, "resources/codegen_templates/go/go_srv_code.txt")
}

func GenerateGOClient(srvDef *SrvDef) ([]byte, error) {

	name := srvDef.Name
	clientname := name + "client"
	return GenerateFromEmbeddTemplate(srvDef, clientname, "resources/codegen_templates/go/go_client_code.txt")
}

func GenerateGOMod(srvDef *SrvDef) ([]byte, error) {

	name := srvDef.Name
	srvname := name + "srv"
	return GenerateFromEmbeddTemplate(srvDef, srvname, "resources/codegen_templates/go/go_srv_mod.txt")
}

func GenerateGOAPI(srvDef *SrvDef) ([]byte, error) {
	srvname := srvDef.Name + "api"
	return GenerateFromEmbeddTemplate(srvDef, srvname, "resources/codegen_templates/go/go_api_code.txt")
}

func prepareTemplateDataMap(srvDef *SrvDef) map[string]interface{} {
	name := srvDef.Name
	srvname := name + "srv"
	namespace := srvDef.Namespace

	base := srvDef.Base

	commands := srvDef.Commands

	data := make(map[string]interface{})
	data["NAME"] = name
	data["NS"] = namespace
	data["SRVNAME"] = srvname
	data["CLTNAME"] = name + "client"
	data["APINAME"] = name + "api"
	data["CMDS"] = commands
	data["BASE"] = base
	data["SPEC"] = srvDef

	//if go_package is set in the spec file
	//use it to set the go package and import
	//otherwise use the name of the service
	//as the package name. only if the go_package
	//is set with 'importpath;packagename'
	goPkg, ok := srvDef.GenOpts["go_package"]
	if ok {
		idx := strings.Index(goPkg, ";")
		if idx > 0 {
			data["GOPKG"] = goPkg[idx+1:]
			data["GOIMPORT"] = goPkg[:idx]
		} else {
			data["GOIMPORT"] = goPkg
			data["GOPKG"] = name
		}
	} else {
		data["GOPKG"] = name
	}

	return data
}
