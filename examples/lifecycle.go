package main

import (
	"context"
	"fmt"
	"mrz.io/itermctl/pkg/itermctl"
	iterm2 "mrz.io/itermctl/pkg/itermctl/proto"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cookie, err := itermctl.GetCookieAndKey("itermctl_lifecycle_example", true)
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
		conn.Close()
		cancel()
	}()

	terminatedSessions, err := itermctl.TerminateSessionMonitor(nil, client)
	if err != nil {
		panic(err)
	}

	newSessions, err := itermctl.NewSessionMonitor(nil, client)
	if err != nil {
		panic(err)
	}

	timed, _ := context.WithTimeout(ctx, 2*time.Second)

	prompts, err := itermctl.PromptMonitor(
		timed,
		client,
		iterm2.PromptMonitorMode_COMMAND_START,
		iterm2.PromptMonitorMode_COMMAND_END,
		iterm2.PromptMonitorMode_PROMPT,
	)

	if err != nil {
		panic(err)
	}

	go func() {
		for ts := range terminatedSessions {
			fmt.Printf("%s: terminated\n", ts)
		}
	}()

	go func() {
		for ns := range newSessions {
			fmt.Printf("%s: started\n", ns)
		}
	}()

	go func() {
		for p := range prompts {
			fmt.Printf("%s: prompt type=%s, ID=%s, \n", p.GetSession(), p.GetEvent(), p.GetUniquePromptId())
		}
	}()

	<-ctx.Done()
}
