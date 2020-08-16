package main

import (
	"github.com/spf13/cobra"
	"mrz.io/itermctl/pkg/itermctl/cmd"
	"os"
)

func main() {
	rootCommand := &cobra.Command{
		Use: os.Args[0],
	}

	rootCommand.AddCommand(
		cmd.RpcCommand.AsCobraCommand(),
		cmd.AutolaunchCommand,
	)

	if err := rootCommand.Execute(); err != nil {
		os.Exit(1)
	}
}
