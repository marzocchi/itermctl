package cli

import (
	"context"
	"github.com/spf13/cobra"
	"mrz.io/itermctl/pkg/itermctl"
	"os"
	"os/signal"
	"syscall"
)

type NestedCommand struct {
	*cobra.Command
	Subcommands []*NestedCommand
}

func (nc *NestedCommand) AsCobraCommand() *cobra.Command {
	cobraCmd := nc.Command
	for _, subc := range nc.Subcommands {
		cobraCmd.AddCommand(subc.AsCobraCommand())
	}

	return cobraCmd
}

func RunWithCtx(f func(ctx context.Context, cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-signals
		cancel()
	}()

	return func(cmd *cobra.Command, args []string) error {
		return f(ctx, cmd, args)
	}
}

func RunWithClient(appName string, f func(client *itermctl.Client, cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return RunWithCtx(func(ctx context.Context, cmd *cobra.Command, args []string) error {
		activate, err := cmd.Flags().GetBool("activate")
		if err != nil {
			return err
		}

		credentials, err := itermctl.GetCredentials(appName, activate)
		if err != nil {
			return err
		}

		conn, err := itermctl.Connect(appName, credentials)
		if err != nil {
			return err
		}

		client := itermctl.NewClient(conn)

		defer conn.Close()

		return f(client, cmd, args)
	})
}
