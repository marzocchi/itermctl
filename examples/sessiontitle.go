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

	tp := itermctl.TitleProvider{
		DisplayName: "Example Title Provider",
		Identifier:  "io.mrz.itermctl.example.title-provider",
		Rpc: itermctl.Rpc{
			Name: "itermctl_example_title_provider",
			Args: args,
			F: func(invocation *itermctl.RpcInvocation) (interface{}, error) {
				err := invocation.Args(&args)
				if err != nil {
					return nil, err
				}

				return fmt.Sprintf("Title for session %q", args.SessionId), nil
			},
		},
	}

	err = conn.RegisterSessionTitleProvider(context.Background(), tp)
	if err != nil {
		panic(err)
	}

	conn.Wait()
}
