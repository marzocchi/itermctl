package main

import (
	"context"
	"fmt"
	"mrz.io/itermctl"
	"mrz.io/itermctl/rpc"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	conn, err := itermctl.GetCredentialsAndConnect("itermctl_rpc_example", true)
	if err != nil {
		panic(err)
	}

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signals
		conn.Close()
	}()

	err = rpc.Register(context.Background(),
		conn,
		rpc.RPC{
			Name:     "itermctl_example_say",
			Args:     SayArgs{Message: "hello, world!", SessionId: "id"},
			Function: Say,
		})

	if err != nil {
		panic(err)
	}

	conn.Wait()
}

type SayArgs struct {
	Message   string
	SessionId string `arg.ref:"id"`
}

func Say(invocation *rpc.Invocation) (interface{}, error) {
	args := SayArgs{}
	err := invocation.Args(&args)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("say")
	cmd.Stdin = strings.NewReader(fmt.Sprintf("session %s has a message: %s", args.SessionId, args.Message))
	err = cmd.Start()
	return nil, err
}
