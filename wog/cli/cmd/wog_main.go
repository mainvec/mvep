package main

import (
	"context"
	"fmt"

	wogcli "github.com/workoak/wop/wog/cli"
	"github.com/workoak/woutil/cli"
)

func main() {
	app := NewCli()
	err := app.Run(context.Background())
	if err != nil {
		panic(err)
	}
}

func NewCli() *cli.Framework {
	app := &cli.Framework{}

	rootCmd := &cli.Command{
		Usage:   "wog",
		Short:   "generate artifacts based on wop schema",
		Version: "v0.1",
		// Run: func(ctx *cli.Context, args []string) {

		// },
	}

	prepareInitCmd(rootCmd)
	prepareGenerateCmd(rootCmd)
	app.Root = rootCmd

	return app
}

func prepareInitCmd(rootCmd *cli.Command) {
	initCmd := &wogcli.InitializeCmd{}

	initCli := &cli.Command{
		Usage: "init",
		Short: "initalize a new wop schema",
		Run: func(ctx *cli.Context, args []string) {
			_, err := wogcli.RunInitializeCmd(ctx, initCmd)
			if err != nil {
				fmt.Fprintln(ctx.Errout(), err)

			}
		},
	}

	initCli.Flags().StringVar(&initCmd.Name, "name", "", "name of the wop schema")
	initCli.Flags().StringVar(&initCmd.Name, "namespace", "", "namespace of the wop schema")
	rootCmd.AddCommand(initCli)
}

func prepareGenerateCmd(rootCmd *cli.Command) {
	generateCmd := &wogcli.GenerateCmd{}
	generateCli := &cli.Command{
		Usage: "generate",
		Short: "generate code artifacts based on wop schema",
		Run: func(ctx *cli.Context, args []string) {
			_, err := wogcli.RunGenerateCmd(ctx, generateCmd)
			if err != nil {
				fmt.Fprintln(ctx.Errout(), err)

			}
		},
	}

	generateCli.Flags().StringVar(&generateCmd.In, "in", "", "input wop schema file")
	generateCli.Flags().StringVar(&generateCmd.Lang, "lang", "", "language to generate code artifacts. e.g. go")
	generateCli.Flags().StringVar(&generateCmd.Outdir, "outdir", "", "output directory for generated code artifacts")

	rootCmd.AddCommand(generateCli)

}
