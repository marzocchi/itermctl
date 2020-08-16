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
	conn, err := itermctl.GetCredentialsAndConnect("itermctl_screenstreamer_example", true)
	if err != nil {
		panic(err)
	}

	client := itermctl.NewClient(conn)
	app := itermctl.NewApp(client)

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signals
		conn.Close()
	}()

	sessionId, err := app.ActiveSessionId()
	if err != nil {
		panic(err)
	}

	screenUpdates, err := itermctl.MonitorScreenUpdates(context.Background(), client, sessionId)
	if err != nil {
		panic(err)
	}

	go func() {
		for su := range screenUpdates {
			fmt.Printf("screen updated in session %s:\n", su.GetSession())
			contents, err := app.ScreenContents(su.GetSession())
			if err != nil {
				panic(err)
			}

			fmt.Printf("%#v", contents)
		}
	}()

	<-conn.Done()
}
