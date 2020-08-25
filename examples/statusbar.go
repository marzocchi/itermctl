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

var Clock = itermctl.StatusBarComponent{
	ShortDescription: "Example UpdateClock",
	Description:      "Example UpdateClock",
	Exemplar:         "2020-08-16 19:30:08 +0200 CEST",
	UpdateCadence:    1,
	Identifier:       "io.mrz.itermctl.example.clock",
	Knobs:            ClockKnobs{Location: "UTC"},
	OnClick:          OnClick,
	Rpc: itermctl.Rpc{
		Name: "itermctl_example_clock",
		F:    UpdateClock,
	},
}

type ClockKnobs struct {
	Location string
}

func UpdateClock(invocation *itermctl.RpcInvocation) (interface{}, error) {
	knobs := &ClockKnobs{}
	err := invocation.Knobs(knobs)
	if err != nil {
		return nil, err
	}

	location, err := time.LoadLocation(knobs.Location)

	now := time.Now().In(location)
	return fmt.Sprintf("%s", now.Round(1*time.Second)), nil
}

func OnClick(invocation *itermctl.RpcInvocation) (interface{}, error) {
	args := itermctl.ClickArgs{}
	if err := invocation.Args(&args); err != nil {
		return nil, fmt.Errorf("click handler: %w", err)
	}

	html := fmt.Sprintf("<p>hello session: %s</p>", args.SessionId)

	if err := invocation.OpenPopover(html, 320, 240); err != nil {
		return nil, fmt.Errorf("click handler: %w", err)
	}

	return nil, nil
}

func main() {
	conn, err := itermctl.GetCredentialsAndConnect("itermctl_statusbar_example", true)
	if err != nil {
		panic(err)
	}

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signals
		conn.Close()
	}()

	err = conn.RegisterStatusBarComponent(context.Background(), Clock)
	if err != nil {
		panic(err)
	}

	conn.Wait()
}
