package itermctl

import (
	"context"
	"fmt"
	"mrz.io/itermctl/iterm2"
)

// MonitorPrompts subscribe to PromptNotification for the given modes, and writes them to the returned channel, until
// the given context is done or the Connection is shutdown. Note that iTerm2 can only detect prompts when shell
// integration is installed.
// See https://iterm2.com/python-api/prompt.html#iterm2.PromptMonitor.
func MonitorPrompts(ctx context.Context, conn *Connection, sessionId string, modes ...iterm2.PromptMonitorMode) (<-chan *iterm2.PromptNotification, error) {
	if len(modes) == 0 {
		modes = []iterm2.PromptMonitorMode{
			iterm2.PromptMonitorMode_COMMAND_START,
			iterm2.PromptMonitorMode_COMMAND_END,
			iterm2.PromptMonitorMode_PROMPT,
		}

	}
	req := NewNotificationRequest(true, iterm2.NotificationType_NOTIFY_ON_PROMPT, sessionId)
	req.Arguments = &iterm2.NotificationRequest_PromptMonitorRequest{
		PromptMonitorRequest: &iterm2.PromptMonitorRequest{
			Modes: modes,
		},
	}

	recv, err := conn.Subscribe(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("prompt monitor: %w", err)
	}

	prompts := make(chan *iterm2.PromptNotification)

	go func() {
		for msg := range recv.Ch() {
			if msg.GetNotification().GetPromptNotification() != nil {
				prompts <- msg.GetNotification().GetPromptNotification()
			}
		}
		close(prompts)
	}()

	return prompts, nil
}
