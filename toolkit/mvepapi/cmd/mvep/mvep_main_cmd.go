// NOMVGEN
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mainvec/mvep/toolkit"
	"github.com/mainvec/ugo/cli"
)

func main() {
	app := NewCli()
	err := app.Run(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func NewCli() *cli.Framework {
	app := &cli.Framework{}

	rootCmd := &cli.Command{
		Usage:   "mvep",
		Short:   "Mainvec Platform Toolkit",
		Long:    "Command-line toolkit for Mainvec Platform",
		Version: MVEPVersion(),
	}

	prepareGenerateCmd(rootCmd)
	prepareGenCmd(rootCmd)
	prepareInitializeCmd(rootCmd)
	prepareValidateCmd(rootCmd)
	app.Root = rootCmd

	return app
}

func prepareGenerateCmd(rootCmd *cli.Command) {
	var formatFlag string
	var inFlag string
	var langFlag string
	var outdirFlag string

	cliCmd := &cli.Command{
		Usage: "generate",
		Short: "Generate code from MVP schema",
		Run: func(ctx *cli.Context, args []string) {
			err := toolkit.ExecuteGenerate(ctx, inFlag, outdirFlag, langFlag, false, formatFlag)
			if err != nil {
				fmt.Fprintln(ctx.Errout(), err)
			}
		},
	}
	cliCmd.Flags().StringVar(&formatFlag, "format", "plain", "serialization format (plain=plain json, pb3=protobuf)")
	cliCmd.Flags().StringVar(&inFlag, "in", "", "input mvep spec file")
	cliCmd.Flags().StringVar(&langFlag, "lang", "", "")
	cliCmd.Flags().StringVar(&outdirFlag, "outdir", "", "")
	rootCmd.AddCommand(cliCmd)

}

func prepareGenCmd(rootCmd *cli.Command) {
	var formatFlag string
	var inFlag string
	var langFlag string
	var outdirFlag string

	cliCmd := &cli.Command{
		Usage: "gen",
		Short: "Generate code from MVP schema (alias)",
		Run: func(ctx *cli.Context, args []string) {
			err := toolkit.ExecuteGenerate(ctx, inFlag, outdirFlag, langFlag, false, formatFlag)
			if err != nil {
				fmt.Fprintln(ctx.Errout(), err)
			}
		},
	}
	cliCmd.Flags().StringVar(&formatFlag, "format", "plain", "serialization format (plain=plain json, pb3=protobuf)")
	cliCmd.Flags().StringVar(&inFlag, "in", "", "input mvep spec file")
	cliCmd.Flags().StringVar(&langFlag, "lang", "", "")
	cliCmd.Flags().StringVar(&outdirFlag, "outdir", "", "")
	rootCmd.AddCommand(cliCmd)

}

func prepareInitializeCmd(rootCmd *cli.Command) {
	var nameFlag string
	var namespaceFlag string

	cliCmd := &cli.Command{
		Usage: "init",
		Short: "Initialize a new MVP spec",
		Run: func(ctx *cli.Context, args []string) {
			err := toolkit.ExeucuteInitializeCmd(ctx, nameFlag, namespaceFlag)
			if err != nil {
				fmt.Fprintln(ctx.Errout(), err)
			}
		},
	}
	cliCmd.Flags().StringVar(&nameFlag, "name", "", "")
	cliCmd.Flags().StringVar(&namespaceFlag, "ns", "", "")
	rootCmd.AddCommand(cliCmd)

}

func prepareValidateCmd(rootCmd *cli.Command) {
	var inFlag string

	cliCmd := &cli.Command{
		Usage: "validate",
		Short: "Validate MVP spec",
		Run: func(ctx *cli.Context, args []string) {
			res, err := toolkit.ExecuteValidateCmd(ctx, inFlag)
			if err != nil {
				fmt.Fprintln(ctx.Errout(), err)
				return
			}

			if !res.Valid() {
				errs := make([]string, 0, len(res.ValidationErrors()))
				for _, e := range res.ValidationErrors() {
					errs = append(errs, e.String())
				}
				fmt.Fprintf(os.Stderr, "MVP Spec is invalid: %v\n", errs)
				return
			}

			fmt.Println("valid!.")
		},
	}
	cliCmd.Flags().StringVar(&inFlag, "in", "", "input mvep spec file")
	rootCmd.AddCommand(cliCmd)

}
