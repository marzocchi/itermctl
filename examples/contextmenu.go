package main

import (
	"context"
	"fmt"
	"mrz.io/itermctl/pkg/itermctl"
	"mrz.io/itermctl/pkg/itermctl/rpc"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	conn, err := itermctl.GetCredentialsAndConnect("itermctl_context_menu_provider_example", true)
	if err != nil {
		panic(err)
	}

	app, err := itermctl.NewApp(conn)
	if err != nil {
		panic(err)
	}

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signals
		conn.Close()
	}()

	var args struct {
		SessionId string `arg.name:"session_id" arg.ref:"id"`
	}

	cm := rpc.ContextMenuProvider{
		DisplayName: "Example Context Menu Provider",
		Identifier:  "io.mrz.itermctl.example.context-menu-provider",
		RPC: rpc.RPC{
			Name: "itermctl_example_context_menu_provider",
			Args: args,
			Function: func(invocation *rpc.Invocation) (interface{}, error) {
				err := invocation.Args(&args)
				if err != nil {
					return nil, err
				}

				text, err := app.Session(args.SessionId).SelectedText()
				if err != nil {
					panic(err)
				}

				fmt.Printf("selected text:\n---\n%s\n---\n", text)
				return nil, nil
			},
		},
	}

	err = rpc.RegisterContextMenuProvider(context.Background(), conn, cm)
	if err != nil {
		panic(err)
	}

	conn.Wait()
}
