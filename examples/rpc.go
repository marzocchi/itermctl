package main

import (
	"context"
	"mrz.io/itermctl/pkg/itermctl"
	"mrz.io/itermctl/pkg/itermctl/rpc"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	cookie, err := itermctl.GetCookieAndKey("itermctl_rpc_example", true)
	if err != nil {
		panic(err)
	}

	conn, err := itermctl.Connect(cookie)
	if err != nil {
		panic(err)
	}

	client := itermctl.NewClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signals
		cancel()
		conn.Close()
	}()

	// this can be called from eg. a keybinding as
	// itermctl_example_say(message: "hello, world")
	callback := rpc.NewRPC(
		"itermctl_example_say",
		func(args *rpc.Invocation) (interface{}, error) {
			msg, err := args.GetString("message")
			if err != nil {
				return nil, err
			}

			cmd := exec.Command("say")
			cmd.Stdin = strings.NewReader(msg)
			err = cmd.Run()
			return nil, err
		},
		rpc.Arg{Name: "message"},
	)

	err = rpc.Register(nil, client, callback)
	if err != nil {
		panic(err)
	}

	<-ctx.Done()
}
