// +build test_with_iterm

package integration_test

import (
	"context"
	"mrz.io/itermctl/pkg/itermctl"
	"mrz.io/itermctl/pkg/itermctl/internal/test"
	iterm2 "mrz.io/itermctl/pkg/itermctl/proto"
	"reflect"
	"testing"
	"time"
)

func TestPromptMonitor(t *testing.T) {
	conn, err := itermctl.GetCredentialsAndConnect(test.AppName(t), true)
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app, err := itermctl.NewApp(conn)
	if err != nil {
		t.Fatal(err)
	}

	promptNotifications, err := conn.MonitorPrompts(ctx)
	if err != nil {
		t.Fatal(err)
	}

	testWindowResp, err := app.CreateTab("", 0, "")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = app.CloseWindow(true, testWindowResp.GetWindowId())
		if err != nil {
			t.Fatal(err)
		}
	}()

	session, err := app.Session(testWindowResp.GetSessionId())
	if err != nil {
		t.Fatal(err)
	}

	if err := session.SendText("ls\n\n", false); err != nil {
		t.Fatal(err)
	}

	prompts := collectPrompts(promptNotifications, 3, t)

	if findPrompt(prompts, &iterm2.PromptNotification_Prompt{}) == nil {
		t.Fatal("expected a PromptNotification_Prompt, got nil")
	}

	if findPrompt(prompts, &iterm2.PromptNotification_CommandEnd{}) == nil {
		t.Fatal("expected a PromptNotification_CommandEnd, got nil")
	}

	if findPrompt(prompts, &iterm2.PromptNotification_CommandStart{}) == nil {
		t.Fatal("expected a PromptNotification_CommandStart, got nil")
	}
}

func findPrompt(prompts []*iterm2.PromptNotification, event interface{}) *iterm2.PromptNotification {
	for _, p := range prompts {
		if reflect.TypeOf(p.GetEvent()) == reflect.TypeOf(event) {
			return p
		}
	}

	return nil
}

func collectPrompts(src <-chan *iterm2.PromptNotification, max int, t *testing.T) []*iterm2.PromptNotification {
	var prompts []*iterm2.PromptNotification

	for {
		select {
		case <-time.After(2 * time.Second):
			t.Fatal("timed out")
		case prompt := <-src:
			prompts = append(prompts, prompt)
			if len(prompts) == max {
				return prompts
			}
		}
	}
	return nil
}
