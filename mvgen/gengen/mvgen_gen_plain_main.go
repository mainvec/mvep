package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/mainvec/mvep/mvgen"
)

// MVGEN plain-format self-generator. to be run manually when needed.

func main() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	in := filepath.Join(wd, "mvgen_plain.jsonc")
	lang := "go"
	outdir := filepath.Join(wd, "..", "mvpapi")
	err = mvgen.ExecuteGenerate(context.Background(), in, outdir, lang, false, "plain")
	if err != nil {
		log.Fatalf("error executing plain generate: %v", err)
	}
}
