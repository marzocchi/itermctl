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

func WithContext(f func(ctx context.Context, cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
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

func WithApp(appName string, f func(app *itermctl.App, cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return WithConnection(appName, func(conn *itermctl.Connection, cmd *cobra.Command, args []string) error {
		app, err := itermctl.NewApp(conn)
		if err != nil {
			return err
		}

		return f(app, cmd, args)
	})
}

func WithConnection(appName string, f func(conn *itermctl.Connection, cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return WithContext(func(ctx context.Context, cmd *cobra.Command, args []string) error {
		activate, err := cmd.Flags().GetBool("activate")
		if err != nil {
			return err
		}

		conn, err := itermctl.GetCredentialsAndConnect(appName, activate)
		if err != nil {
			return err
		}

		defer conn.Close()

		return f(conn, cmd, args)
	})
}
