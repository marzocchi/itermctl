package main

import (
	"context"
	"fmt"
	"mrz.io/itermctl"
	"mrz.io/itermctl/rpc"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	conn, err := itermctl.GetCredentialsAndConnect("itermctl_titleprovider_example", true)
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

	tp := rpc.TitleProvider{
		DisplayName: "Example Title Provider",
		Identifier:  "io.mrz.itermctl.example.title-provider",
		RPC: rpc.RPC{
			Name: "itermctl_example_title_provider",
			Args: args,
			Function: func(invocation *rpc.Invocation) (interface{}, error) {
				err := invocation.Args(&args)
				if err != nil {
					return nil, err
				}

				return fmt.Sprintf("Title for session %q", args.SessionId), nil
			},
		},
	}

	err = rpc.RegisterSessionTitleProvider(context.Background(), conn, tp)
	if err != nil {
		panic(err)
	}

	conn.Wait()
}
