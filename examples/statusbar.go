package main

import (
	"context"
	"fmt"
	"mrz.io/itermctl/pkg/itermctl"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	conn, err := itermctl.GetCredentialsAndConnect("itermctl_statusbar_example", true)
	if err != nil {
		panic(err)
	}

	client := itermctl.NewClient(conn)

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signals
		conn.Close()
	}()

	cmp := itermctl.StatusBarComponent{
		ShortDescription: "Example Clock",
		Description:      "Example Clock",
		Exemplar:         "2020-08-16 19:30:08 +0200 CEST",
		UpdateCadence:    1,
		Identifier:       "io.mrz.itermctl.example.clock",
		Rpc: itermctl.Rpc{
			Name: "itermctl_example_clock",
			F:    Clock,
		},
		Knobs: ClockKnobs{Location: "UTC"},
	}

	err = itermctl.RegisterStatusBarComponent(context.Background(), client, cmp)
	if err != nil {
		panic(err)
	}

	<-conn.Done()
}

type ClockKnobs struct {
	Location string
}

func Clock(args *itermctl.RpcInvocation) (interface{}, error) {
	knobs := &ClockKnobs{}
	err := args.Knobs(knobs)
	if err != nil {
		return nil, err
	}

	location, err := time.LoadLocation(knobs.Location)

	now := time.Now().In(location)
	return fmt.Sprintf("%s", now.Round(1*time.Second)), nil
}
