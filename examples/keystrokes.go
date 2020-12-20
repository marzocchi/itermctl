package main

import (
	"context"
	"fmt"
	"mrz.io/itermctl"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	conn, err := itermctl.GetCredentialsAndConnect("itermctl_keystrokes_example", true)
	if err != nil {
		panic(err)
	}

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signals
		conn.Close()
	}()

	keystrokes, err := itermctl.MonitorKeystrokes(context.Background(), conn, itermctl.AllSessions)
	if err != nil {
		panic(err)
	}

	go func() {
		for ks := range keystrokes {
			fmt.Printf("typed: %s\n", ks.GetCharacters())
		}
	}()

	conn.Wait()
}
