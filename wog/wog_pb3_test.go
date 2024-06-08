package wog_test

import (
	"log"
	"os"
	"testing"

	"github.com/workoak/wop/wog"
)

// Protobuf Def test data
var tests = []struct {
	name          string
	testfile_path string
	wantPB3       string
	wantErr       bool
	numOfErrors   int
}{
	{"TEST01 :Valid - basic without comments", "01_basic_wo_valid_no_comments.json",
		`syntax = "proto3";
		package test1Namespace.test1Name;
	`,
		false, 0},
	{"TEST04 :Valid - basic cmd", "04_valid_wosrv_command.jsonc",
		`
		syntax = "proto3";
		package test4Namespace.test4Name;
		import "google/protobuf/duration.proto";
		import "google/protobuf/timestamp.proto";

		option go_package = "github.com/workoak/wo/wop/wopdb/wopdb2api";
		
		message OrderPizzaCmd {
		}
		message OrderPizzaCmdResult {
		}
	`,
		false, 0},
	{"TEST05 :Valid - cmd with fields", "05_command_withfields.jsonc",
		`
		syntax = "proto3";
		package test5Namespace.test5Name;
		import "google/protobuf/duration.proto";
		import "google/protobuf/timestamp.proto";

		message OrderPizzaCmd {
			int32 size = 1;
			string type = 2;
			repeated string toppings = 3;
			
		}
		message OrderPizzaCmdResult {
		}
	`,
		false, 0},
	{"TEST06 :Valid/comands with refs", "06_command_with_ref.jsonc",
		`
		syntax = "proto3";
		package test6Namespace.test6Name;
		import "google/protobuf/duration.proto";
		import "google/protobuf/timestamp.proto";

		option go_package = "github.com/workoak/wo/wop/wopdb/wopdb2api";
		
		message Address {
			string street = 1;
		      
			string city = 2;
		      
			string country = 3;
		      }

		message OrderPizzaCmd {
			string size = 1;
			string type = 2;
			repeated string toppings = 3;
			Address address = 4;
			
		}
		message OrderPizzaCmdResult {
		}
	`,
		false, 0},
	{"TEST08 :Valid/map fields", "08_maps.jsonc",
		`
		syntax = "proto3";
		package test8.mapstest;
		import "google/protobuf/duration.proto";
		import "google/protobuf/timestamp.proto";

		option go_package = "github.com/workoak/wop/won/mapstest";
		
		message WOReqTest {
			string woservs = 1;
		      
			map<string, string> headers = 2;
		      }
	`,
		false, 0},
}

func TestBuildProtoBuffDefFromJSON(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd, err := getTestFilePath(tt.testfile_path)
			if err != nil {
				log.Fatal(err.Error())
			}
			specfile, err := os.Open(wd)
			if err != nil {
				t.Errorf("error reading test file %v,%e", tt.testfile_path, err)
				return
			}
			defer specfile.Close()
			if err != nil {
				log.Fatalf("error reading test file %v,%e", tt.testfile_path, err)
			}

			result, err := wog.BuildProtoBuffDefFromJSON(specfile)
			if (err != nil) != tt.wantErr {
				t.Errorf(" error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			wantPb3Descrp, err := wog.ParseProto3Definition(string(result.Name()), []byte(tt.wantPB3))

			if err != nil {
				log.Fatal(err.Error())
			}

			if !wog.FileDescriptorEqual(result, wantPb3Descrp) {
				//println("[======")
				//GenerateProtobuf3FromFileDesc(result, os.Stdout)
				//println("======]")
				t.Errorf("BuildProtoBuffDefFromJSON: ==GOT %v\n==WANTED %v", result, wantPb3Descrp)
				return

			}

		})
	}

}
