package main

import (
	"context"
	"fmt"
	"mrz.io/itermctl/pkg/itermctl"
	iterm2 "mrz.io/itermctl/pkg/itermctl/proto"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	conn, err := itermctl.GetCredentialsAndConnect("itermctl_lifecycle_example", true)
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

	terminatedSessions, err := itermctl.MonitorSessionsTermination(context.Background(), client)
	if err != nil {
		panic(err)
	}

	newSessions, err := itermctl.MonitorNewSessions(context.Background(), client)
	if err != nil {
		panic(err)
	}

	prompts, err := itermctl.MonitorPrompts(
		context.Background(),
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
			fmt.Printf("%s: terminated\n", ts.GetSessionId())
		}
	}()

	go func() {
		for ns := range newSessions {
			fmt.Printf("%s: started\n", ns.GetSessionId())
		}
	}()

	go func() {
		for p := range prompts {
			fmt.Printf("%s: prompt type=%s, ID=%s, \n", p.GetSession(), p.GetEvent(), p.GetUniquePromptId())
		}
	}()

	<-conn.Done()
}
