package mvgen_test

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/mainvec/mvep/mvgen"
)

func TestGenerateGOProtoBuffAPI(t *testing.T) {
	tests := []struct {
		name          string
		testfile_path string
		want          bool
		wantErr       bool
	}{
		{"TEST04 :Valid/basic cmd", "04_valid_wosrv_command.jsonc", true, false},
		{"TEST05 :InValid/no go_package, ", "05_command_withfields.jsonc", true, true},
		{"TEST06 :Valid/comands with refs", "06_command_with_ref.jsonc", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd, err := getTestFilePath(tt.testfile_path)
			if err != nil {
				log.Fatal(err.Error())
			}
			specfile, err := os.Open(wd)
			if err != nil {

			}
			defer specfile.Close()
			if err != nil {
				log.Fatalf("error reading test file %v,%e", tt.testfile_path, err)
			}

			srvDef, err := mvgen.BuildSrvDefFromJSON(specfile)
			if err != nil {
				log.Fatalf("error building SrvDef %v,%e", tt.testfile_path, err)
			}

			result, err := mvgen.BuildProtoBuffDefFromSrvDef(srvDef)

			if err != nil {
				log.Fatalf("error BuildProtoBuffDefFromJSON test file %v,%e", tt.testfile_path, err)
			}
			buff := &bytes.Buffer{}
			err = mvgen.GenerateProtobuf3FromFileDesc(result, buff)
			if err != nil {
				log.Fatalf("error GenerateProtobuf3FromFileDesc test file %v,%e", tt.testfile_path, err)
			}

			pb3GOAPI, err := mvgen.GenerateGOProtoBuffAPIFromProto(srvDef, buff.Bytes())
			if (err != nil) && !tt.wantErr {
				t.Errorf("got error[%v], wanted error[%v], error[%v]", err != nil, tt.wantErr, err)
				return
			}

			if err == nil && len(pb3GOAPI) == 0 {
				t.Errorf(" no GO Pb3 API generated ")
				return
			}
			fmt.Printf("Golang==== \n%v\n====", string(pb3GOAPI))
		})
	}
}

func TestGenerateGOSRV(t *testing.T) {
	t.Skip()
	tests := []struct {
		name          string
		testfile_path string
		want          bool
		wantErr       bool
	}{
		{"TEST07 :Valid/pizzahub", "07_pizzahub.jsonc", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd, err := getTestFilePath(tt.testfile_path)
			if err != nil {
				log.Fatal(err.Error())
			}
			specfile, err := os.Open(wd)
			if err != nil {
				log.Fatalf("error reading test file %v,%e", tt.testfile_path, err)
			}
			defer specfile.Close()
			srvDef, err := mvgen.BuildSrvDefFromJSON(specfile)
			if err != nil {
				log.Fatalf("error BuildSrvDefFromJSON test file %v,%e", tt.testfile_path, err)
			}

			//generate GO MOD
			goMod, err := mvgen.GenerateGOMod(srvDef)
			if err == nil && len(goMod) == 0 {
				t.Errorf(" no GO goMod generated ")
				return
			}
			saveTestTempFile(srvDef, "go", "go.mod", goMod)

			//generate .proto
			buff := &bytes.Buffer{}
			mvgen.GenerateProtobuf3FromSrvDef(srvDef, buff)
			saveTestTempFile(srvDef, "model", srvDef.Name+".proto", buff.Bytes())

			//generate GO PB3 API
			pb3GOAPI, err := mvgen.GenerateGOProtoBuffAPIFromProto(srvDef, buff.Bytes())
			if (err != nil) && !tt.wantErr {
				t.Errorf("got error[%v], wanted error[%v], error[%v]", err != nil, tt.wantErr, err)
				return
			}

			if err == nil && len(pb3GOAPI) == 0 {
				t.Errorf(" no GO Pb3 API generated ")
				return
			}
			saveTestTempFile(srvDef, filepath.Join("go", "api"), srvDef.Name+".pb.go", pb3GOAPI)

			//generate GO SRV
			goSRV, err := mvgen.GenerateGOSRV(srvDef)
			if (err != nil) && !tt.wantErr {
				t.Errorf("got error[%v], wanted error[%v], error[%v]", err != nil, tt.wantErr, err)
				return
			}

			if err == nil && len(goSRV) == 0 {
				t.Errorf(" no GO SRV generated ")
				return
			}
			//saveTestTempFile(srvDef, filepath.Join("go", "srv"), srvDef.Name+".srv.go", goSRV)

			////generate GO Client
			goClient, err := mvgen.GenerateGOClient(srvDef)
			if err != nil {
				log.Fatalf("error GenerateGOClient test file %v,%e", tt.testfile_path, err)
			}

			if err == nil && len(goClient) == 0 {
				t.Errorf(" no GO Client generated ")
				return
			}
			//saveTestTempFile(srvDef, filepath.Join("go", "client"), srvDef.Name+".client.go", goClient)

		})
	}
}

func TestGenerateGOVanillaStructs(t *testing.T) {
	tests := []struct {
		name          string
		testfile_path string
		wantErr       bool
		expectTime    bool // whether time import is expected
	}{
		{"TEST05: Fields with repeated", "05_command_withfields.jsonc", false, false},
		{"TEST06: Commands with refs", "06_command_with_ref.jsonc", false, false},
		{"TEST08: Maps", "08_maps.jsonc", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd, err := getTestFilePath(tt.testfile_path)
			if err != nil {
				t.Fatalf("error getting test file path: %v", err)
			}
			specfile, err := os.Open(wd)
			if err != nil {
				t.Fatalf("error opening test file: %v", err)
			}
			defer specfile.Close()

			srvDef, err := mvgen.BuildSrvDefFromJSON(specfile)
			if err != nil {
				t.Fatalf("error building SrvDef: %v", err)
			}

			vanillaAPI, err := mvgen.GenerateGOVanillaStructs(srvDef)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateGOVanillaStructs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && len(vanillaAPI) == 0 {
				t.Errorf("no vanilla Go structs generated")
				return
			}

			// Print generated code for inspection
			fmt.Printf("Vanilla Go Structs (%s):\n%s\n====\n", tt.name, string(vanillaAPI))

			// Basic validations
			output := string(vanillaAPI)
			if !bytes.Contains(vanillaAPI, []byte("package api")) {
				t.Errorf("generated code should contain 'package api'")
			}
			if !bytes.Contains(vanillaAPI, []byte("json:")) {
				t.Errorf("generated code should contain json tags")
			}

			// Check for time import if expected
			if tt.expectTime && !bytes.Contains(vanillaAPI, []byte("\"time\"")) {
				t.Errorf("expected time import but not found")
			}

			_ = output
		})
	}
}
