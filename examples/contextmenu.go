package main

import (
	"context"
	"fmt"
	"mrz.io/itermctl/pkg/itermctl"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	conn, err := itermctl.GetCredentialsAndConnect("itermctl_context_menu_provider_example", true)
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

	cm := itermctl.ContextMenuProvider{
		DisplayName: "Example Context Menu Provider",
		Identifier:  "io.mrz.itermctl.example.context-menu-provider",
		Rpc: itermctl.Rpc{
			Name: "itermctl_example_context_menu_provider",
			Args: args,
			F: func(invocation *itermctl.RpcInvocation) (interface{}, error) {
				err := invocation.Args(&args)
				if err != nil {
					return nil, err
				}

				fmt.Printf("context menu selected in session %s\n", args.SessionId)

				return nil, nil
			},
		},
	}

	err = conn.RegisterContextMenuProvider(context.Background(), cm)
	if err != nil {
		panic(err)
	}

	conn.Wait()
}
