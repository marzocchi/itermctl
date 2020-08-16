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
	cookie, err := itermctl.GetCookieAndKey("itermctl_titleprovider_example", true)
	if err != nil {
		panic(err)
	}

	conn, err := itermctl.Connect(cookie)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := itermctl.NewClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signals
		cancel()
	}()

	tp := rpc.TitleProvider{
		DisplayName: "Example Title Provider",
		Identifier:  "io.mrz.itermctl.example.title-provider",
		RPC: rpc.NewRPC(
			"itermctl_example_title_provider",
			func(i *rpc.Invocation) (interface{}, error) {
				id, err := i.GetString("id")
				if err != nil {
					return nil, err
				}

				return fmt.Sprintf("%s", id), nil
			},
			rpc.Arg{
				Name:      "id",
				Reference: "id",
			},
		),
	}

	err = rpc.RegisterSessionTitleProvider(ctx, client, tp)
	if err != nil {
		panic(err)
	}

	<-ctx.Done()
}
