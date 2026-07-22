package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/mainvec/mvep/toolkit"
)

// MVGEN plain-format self-generator. to be run manually when needed.

func main() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	in := filepath.Join(wd, "toolkit_plain.jsonc")
	lang := "go"
	outdir := filepath.Join(wd, "..", "mvepapi")
	err = toolkit.ExecuteGenerate(context.Background(), in, outdir, lang, false, "plain")
	if err != nil {
		log.Fatalf("error executing plain generate: %v", err)
	}
}
