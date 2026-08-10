// NOMVEP
// mvep_main_cmd.go is hand-written (not generated) and safe to rewrite.
// T16 (plan 025): mvep's own CLI is dogfooded on the descriptor-driven
// runtime/cli library — cli.New(pkg.Describe(), executor) builds the command
// tree from the PackageDesc, and exits via cli.ExitCode. This is the real
// acceptance test: mvep's CLI is its own first consumer on the same path
// every generated package will use.
package main

import (
	"context"
	"fmt"
	"os"

	mveccli "github.com/mainvec/mvep/runtime/go/mvep/cli"
	mvep "github.com/mainvec/mvep/toolkit/mvepapi"
	api "github.com/mainvec/mvep/toolkit/mvepapi/api"
)

func main() {
	pkg := api.NewPackage()
	runner := mvep.GetCommandRunner()

	// pkg implements mvep.PackageDescriber via NewPackageFromDesc (T3).
	// cli.New needs the *PackageDesc to build the command tree.
	desc := api.Describe()
	app := mveccli.New(desc, &mveccli.LocalExecutor{Runner: runner})
	app.Root().Version = MVEPVersion()

	_ = pkg // pkg is available for NameOf/InstanceOf if needed

	err := app.Run(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(mveccli.ExitCode(err))
	}
}
