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

	app, err := itermctl.NewApp(conn)
	if err != nil {
		panic(err)
	}

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signals
		conn.Close()
	}()

	session, err := app.ActiveSession()
	if err != nil {
		panic(err)
	}

	screenUpdates, err := conn.MonitorScreenUpdates(context.Background(), session.Id())
	if err != nil {
		panic(err)
	}

	go func() {
		var lastOffset int32

		for range screenUpdates {
			contents, err := session.ScreenContents()
			if err != nil {
				panic(err)
			}

			fmt.Printf("last offset: %d\n", lastOffset)

			for i, line := range contents.GetContents() {
				fmt.Printf("line %d: %s\n", i, line.GetText())
			}
		}
	}()

	conn.Wait()
}
