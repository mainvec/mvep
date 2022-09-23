package wop_test

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/workoak/wop"
)

func TestIterateByKey(t *testing.T) {
	type OData wop.OMap[string, string]
	testMap := OData{
		"a": "aVal",
		"x": "xVal",
		"c": "cVal",
		"b": "bVal",
	}
	want := []string{"aVal", "bVal", "cVal", "xVal"}
	var got []string
	for iter := wop.IteratorByKey(testMap); iter.HasNext(); {
		_, v := iter.Next()
		got = append(got, v)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v,wanted %v ", got, want)
	}

}

func TestIterateByValue(t *testing.T) {

	type ValStruct struct {
		Fnum  int
		FName string
	}
	type OData map[string]*ValStruct
	testMap := OData{
		"a": &ValStruct{Fnum: 4, FName: "aStruct"},
		"x": &ValStruct{Fnum: 2, FName: "xStruct"},
		"c": &ValStruct{Fnum: 3, FName: "cStruct"},
		"b": &ValStruct{Fnum: 1, FName: "bStruct"},
	}
	t.Run("iterate by fnum order", func(t *testing.T) {
		//order by Fnum
		want := []*ValStruct{testMap["b"], testMap["x"], testMap["c"], testMap["a"]}
		var got []*ValStruct
		lessFunc := func(i, j *ValStruct) bool {
			return i.Fnum < j.Fnum
		}
		for iter := wop.IterateByValue(testMap, lessFunc); iter.HasNext(); {
			_, v := iter.Next()
			got = append(got, v)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v,wanted %v ", got, want)
		}
	})

	t.Run("iterate by fname order", func(t *testing.T) {
		//order by Fnum
		want := []*ValStruct{testMap["a"], testMap["b"], testMap["c"], testMap["x"]}
		var got []*ValStruct
		lessNameFunc := func(i, j *ValStruct) bool {
			return i.FName < j.FName
		}
		for iter := wop.IterateByValue(testMap, lessNameFunc); iter.HasNext(); {
			_, v := iter.Next()
			got = append(got, v)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v,wanted %v ", got, want)
		}
	})

}
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

			result, err := wop.ValidateJSONSchema(specfile)

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
		want          wop.SrvDef
		wantErr       bool
		numOfErrors   int
	}{
		{"TEST01 :Valid - basic ServiceDef", "01_basic_wo_valid_no_comments.json",
			wop.SrvDef{
				Id:        "test1Id",
				Name:      "test1Name",
				Namespace: "test1Namespace",
			}, false, 0},
		{"TEST02 :Valid - basic with comments", "02_basic_wo_valid_with_comments.jsonc",
			wop.SrvDef{
				Id:        "test2Id",
				Name:      "test2Name",
				Namespace: "test2Namespace",
			}, false, 0},
		{"TEST03 :InValid - basic", "03_basic_wo_invalid.jsonc", wop.SrvDef{}, true, 0},
		{"TEST04 :Valid - basic cmd", "04_valid_wosrv_command.jsonc",
			wop.SrvDef{
				Id:        "test4Id",
				Name:      "test4Name",
				Namespace: "test4Namespace",
				GenOpts: wop.GenOptsDef{
					"go_package": "github.com/workoak/wo/wop/wopdb/wopdb2api",
				},
				Commands: wop.CommandDefs{
					"OrderPizzaCmd": wop.CommandDef{
						Title: "Order a pizza",
					},
				},
			}, false, 0},
		{"TEST05 :Valid - cmd with fields", "05_command_withfields.jsonc",
			wop.SrvDef{
				Id:        "test5Id",
				Name:      "test5Name",
				Namespace: "test5Namespace",
				Commands: wop.CommandDefs{
					"OrderPizzaCmd": wop.CommandDef{
						Title: "Order a pizza",
						Fields: wop.FieldDefs{
							"size": {
								Fnum: 1,
								Type: wop.STRING,
							},
							"type": {
								Fnum: 2,
								Type: wop.STRING,
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
			wop.SrvDef{
				Id:        "test6Id",
				Name:      "test6Name",
				Namespace: "test6Namespace",
				GenOpts: wop.GenOptsDef{
					"go_package": "github.com/workoak/wo/wop/wopdb/wopdb2api",
				},
				Commands: wop.CommandDefs{
					"OrderPizzaCmd": wop.CommandDef{
						Title: "Order a pizza",
						Fields: wop.FieldDefs{
							"size": {
								Fnum: 1,
								Type: wop.STRING,
							},
							"type": {
								Fnum: 2,
								Type: wop.STRING,
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
				Records: wop.RecordsDefs{
					"Address": wop.RecordDef{
						Name: "Address",
						Fields: wop.FieldDefs{
							"street": {
								Fnum: 1,
								Type: wop.STRING,
							},
							"city": {
								Fnum: 2,
								Type: wop.STRING,
							},
							"country": {
								Fnum: 3,
								Type: "string",
							},
						},
					},
				},
			}, false, 0},
		{"TEST08 :Valid - map field", "08_maps.jsonc",
			wop.SrvDef{
				Id:        "mapsTest",
				Name:      "mapstest",
				Namespace: "test8",
				Base:      "github.com/workoak/mapstest",
				GenOpts: wop.GenOptsDef{
					"go_package": "github.com/workoak/wop/won/mapstest",
				},
				Records: wop.RecordsDefs{
					"WOReqTest": wop.RecordDef{
						Name: "WOReqTest",
						Fields: wop.FieldDefs{
							"woservs": {
								Fnum: 1,
								Type: wop.STRING,
							},
							"headers": {
								Fnum:         2,
								Type:         wop.MAP,
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

			result, err := wop.BuildSrvDefFromJSON(specfile)

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
			err = wop.MarshalSrvDefToJSON(&buf, result)
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

func saveTestTempFile(srvDef *wop.SrvDef, dir, testfilename string, content []byte) error {
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
