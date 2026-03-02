package protojson_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mainvec/mvp/mvpgo/mvp"
	iunetApi "github.com/mainvec/mvp/mvpgo/test/api"
	_ "github.com/mainvec/mvp/mvpgo/util/protojson"
	enc "github.com/mainvec/ugo/oencoding"
)

func TestProtobuffHttpHandelr(t *testing.T) {

	encoder, ok := enc.LookupEncoding("protojson")
	if !ok {
		t.Fatal("not ok")
	}
	pkg := iunetApi.NewPackage()

	runner := iunetApi.NewCommandRunner()
	runner.RunHubCreateCmd = func(ctx context.Context, cmd *iunetApi.HubCreateCmd) (*iunetApi.HubCreateCmdResult, error) {
		return &iunetApi.HubCreateCmdResult{
			Hub: &iunetApi.Hub{
				Uuid: "uuid",
				Name: cmd.Name,
			},
		}, nil
	}
	pkgHandler := &mvp.PackageHandler{
		Package:       pkg,
		CommandRunner: runner,
	}

	mux := http.NewServeMux()
	cmdPath := "/" + pkg.GetName() + "/cmd"
	mux.Handle("POST "+cmdPath, pkgHandler)

	server := httptest.NewServer(mux)
	defer server.Close()

	cmd := &iunetApi.HubCreateCmd{}
	cmd.Name = "testHub"
	cmd.Broker = "testBroker"

	cmdBytes, err := encoder.Encode(cmd)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("POST", server.URL+cmdPath, bytes.NewBuffer(cmdBytes))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-mainvec-cmd", pkg.NameOf(cmd))
	// Add other headers here

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatal("status code mismatch")
	}

	result := &iunetApi.HubCreateCmdResult{}

	respType := resp.Header.Get("Content-Type")
	if respType != "application/json" {
		t.Fatal("content type mismatch")
	}

	resultByte, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	err = encoder.Decode(resultByte, result)
	if err != nil {
		t.Fatal(err)
	}
	if result.Hub == nil {
		t.Fatal("hub is nil")
	}

	if result.Hub.Uuid != "uuid" {
		t.Fatal("uuid mismatch")
	}

	if result.Hub.Name != "testHub" {
		t.Fatal("name mismatch")
	}

	// handle err and resp
}

func TestMap(t *testing.T) {
	e, ok := enc.LookupEncoding("protojson")
	if !ok {
		t.Fatal("not ok")
	}

	cmd1, ok := iunetApi.InstanceOf("NewIUHubCmd")
	if !ok {
		fmt.Println("not ok")
	}
	cmd1, err := e.Encode(cmd1)
	if err != nil {
		fmt.Println(err)
	}

}
