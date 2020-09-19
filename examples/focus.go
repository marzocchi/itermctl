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
	conn, err := itermctl.GetCredentialsAndConnect("itermctl_focus_example", true)
	if err != nil {
		panic(err)
	}

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signals
		conn.Close()
	}()

	notifications, err := itermctl.MonitorFocus(context.Background(), conn)
	if err != nil {
		panic(err)
	}

	for notification := range notifications {
		fmt.Printf("%s %s\n", notification.Which, notification.Id)
	}
}
