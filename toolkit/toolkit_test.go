package toolkit_test

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mainvec/mvep/toolkit"
)

func TestValidateJSONSchema(t *testing.T) {
	tests := []struct {
		name          string
		testfile_path string
		want          bool
		wantErr       bool
		numOfErrors   int
	}{
		{"TEST01 :Valid - basic without comments", "01_basic_wo_valid_no_comments.json", true, false, 0},
		{"TEST02 :Valid - basic with comments", "02_basic_wo_valid_with_comments.jsonc", true, false, 0},
		{"TEST03 :InValid - basic", "03_basic_wo_invalid.jsonc", false, false, 1},
		{"TEST04 :Valid/basic cmd", "04_valid_wosrv_command.jsonc", true, false, 0},
		{"TEST05 :Valid/comands with fields", "05_command_withfields.jsonc", true, false, 0},
		{"TEST06 :Valid/comands with refs", "06_command_with_ref.jsonc", true, false, 0},
		{"TEST07 :Valid/command with result", "07_command_with_result.jsonc", true, false, 0},
		{"TEST09 :Valid/external schema", "09_external_schema.jsonc", true, false, 0},
		{"TEST10 :Valid/mvpspec external schema", "10_mvpspec_external_schema.jsonc", true, false, 0},
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

			result, err := toolkit.ValidateJSONSchema(specfile)
			if !result.Valid() {
				for _, v := range result.ValidationErrors() {
					fmt.Printf("err:%v", v.String())
				}

			}
			if (err != nil) != tt.wantErr {
				t.Errorf(" error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result.Valid() != tt.want {
				t.Errorf("schema Valid(): got %v, wanted %v, Errors[%v]", result.Valid(), tt.want, result.ValidationErrors())
			}
			if len(result.ValidationErrors()) != tt.numOfErrors {
				t.Errorf("wrong number of errors: got %v, wanted %v", len(result.ValidationErrors()), tt.numOfErrors)
			}
		})
	}
}

func TestBuildSrvDefFromJSON(t *testing.T) {
	tests := []struct {
		name          string
		testfile_path string
		want          toolkit.SrvDef
		wantErr       bool
		numOfErrors   int
	}{
		{"TEST01 :Valid - basic ServiceDef", "01_basic_wo_valid_no_comments.json",
			toolkit.SrvDef{
				Id:        "test1Id",
				Name:      "test1Name",
				Namespace: "test1Namespace",
			}, false, 0},
		{"TEST02 :Valid - basic with comments", "02_basic_wo_valid_with_comments.jsonc",
			toolkit.SrvDef{
				Id:        "test2Id",
				Name:      "test2Name",
				Namespace: "test2Namespace",
			}, false, 0},
		{"TEST03 :InValid - basic", "03_basic_wo_invalid.jsonc", toolkit.SrvDef{}, true, 0},
		{"TEST04 :Valid - basic cmd", "04_valid_wosrv_command.jsonc",
			toolkit.SrvDef{
				Id:        "test4Id",
				Name:      "test4Name",
				Namespace: "test4Namespace",
				GenOpts: toolkit.GenOptsDef{
					"go_package": "github.com/mainvec/wo/mvep/wopdb/wopdb2api",
				},
				Commands: toolkit.CommandDefs{
					"OrderPizzaCmd": toolkit.CommandDef{
						Title: "Order a pizza",
					},
				},
			}, false, 0},
		{"TEST05 :Valid - cmd with fields", "05_command_withfields.jsonc",
			toolkit.SrvDef{
				Id:        "test5Id",
				Name:      "test5Name",
				Namespace: "test5Namespace",
				Commands: toolkit.CommandDefs{
					"OrderPizzaCmd": toolkit.CommandDef{
						Title: "Order a pizza",
						Fields: toolkit.FieldDefs{
							"size": {
								Fnum: 1,
								Type: toolkit.INT32,
							},
							"type": {
								Fnum: 2,
								Type: toolkit.STRING,
							},
							"toppings": {
								Fnum:     3,
								Type:     "string",
								Repeated: true,
							},
						},
					},
				},
			}, false, 0},
		{"TEST06 :Valid/comands with refs", "06_command_with_ref.jsonc",
			toolkit.SrvDef{
				Id:        "test6Id",
				Name:      "test6Name",
				Namespace: "test6Namespace",
				GenOpts: toolkit.GenOptsDef{
					"go_package": "github.com/mainvec/wo/mvep/wopdb/wopdb2api",
				},
				Commands: toolkit.CommandDefs{
					"OrderPizzaCmd": toolkit.CommandDef{
						Title: "Order a pizza",
						Fields: toolkit.FieldDefs{
							"size": {
								Fnum: 1,
								Type: toolkit.STRING,
							},
							"type": {
								Fnum: 2,
								Type: toolkit.STRING,
							},
							"toppings": {
								Fnum:     3,
								Type:     "string",
								Repeated: true,
							},
							"address": {
								Fnum:   4,
								Type:   "recRef",
								RecRef: "#/recordsDefs/Address",
							},
						},
					},
				},
				Records: toolkit.RecordsDefs{
					"Address": toolkit.RecordDef{
						Name: "Address",
						Fields: toolkit.FieldDefs{
							"street": {
								Fnum: 1,
								Type: toolkit.STRING,
							},
							"city": {
								Fnum: 2,
								Type: toolkit.STRING,
							},
							"country": {
								Fnum: 3,
								Type: "string",
							},
						},
					},
				},
			}, false, 0},
		{"TEST07 :Valid/command with result", "07_command_with_result.jsonc",
			toolkit.SrvDef{
				Id:        "test7Id",
				Name:      "test7Name",
				Namespace: "test7Namespace",
				Title:     "test7Title",
				Desc:      "test7Description",
				Version:   "v0.1",
				Commands: toolkit.CommandDefs{
					"OrderPizzaCmd": toolkit.CommandDef{
						Title: "Order a pizza",
						Alias: "order",
						Desc:  "command description",
						Fields: toolkit.FieldDefs{
							"size": {
								Fnum: 1,
								Type: toolkit.STRING,
								Desc: "field description",
							},
						},
						ResultFields: toolkit.FieldDefs{
							"orderStatus": {
								Fnum: 1,
								Type: toolkit.STRING,
							},
						},
					},
				},
			}, false, 0},
		{"TEST08 :Valid - map field", "08_maps.jsonc",
			toolkit.SrvDef{
				Id:        "mapsTest",
				Name:      "mapstest",
				Namespace: "test8",
				Base:      "github.com/mainvec/mapstest",
				GenOpts: toolkit.GenOptsDef{
					"go_package": "github.com/mainvec/mvep/won/mapstest",
				},
				Records: toolkit.RecordsDefs{
					"WOReqTest": toolkit.RecordDef{
						Name: "WOReqTest",
						Fields: toolkit.FieldDefs{
							"woservs": {
								Fnum: 1,
								Type: toolkit.STRING,
							},
							"headers": {
								Fnum:         2,
								Type:         toolkit.MAP,
								MapValueType: "string",
							},
						},
					},
				},
			}, false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd, err := getTestFilePath(tt.testfile_path)
			if err != nil {
				log.Fatal(err.Error())
			}
			specfile, err := os.Open(wd)
			defer specfile.Close()
			if err != nil {
				log.Fatalf("error reading test file %v,%e", tt.testfile_path, err)
			}

			result, err := toolkit.BuildSrvDefFromJSON(specfile)

			if (err != nil) != tt.wantErr {
				t.Errorf(" error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			//not tests if wanted an err
			if tt.wantErr {
				return
			}

			if !reflect.DeepEqual(result, &tt.want) {
				t.Errorf("SrvDef Unmarshal: got %v, wanted %v", result, tt.want)
				return
			}

			//Marsal SrvDef to json and validate
			var buf bytes.Buffer
			err = toolkit.MarshalSrvDefToJSON(&buf, result)
			if err != nil {
				t.Errorf("unexpected error %v", err)
			}

		})
	}
}

func getTestFilePath(testfilename string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	wd = filepath.Join(wd, "testdata", testfilename)
	return wd, nil
}

func saveTestTempFile(srvDef *toolkit.SrvDef, dir, testfilename string, content []byte) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	tempdir := filepath.Join(wd, "test_temp", srvDef.Name, dir)
	err = os.MkdirAll(tempdir, 0755)
	if err != nil {
		return err
	}

	testtemp := filepath.Join(tempdir, testfilename)
	err = os.WriteFile(testtemp, content, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}
