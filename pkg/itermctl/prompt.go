package itermctl

import (
	"context"
	"fmt"
	iterm2 "mrz.io/itermctl/pkg/itermctl/proto"
)

// PromptMonitor subscribe to prompt notifications for the given modes, and writes them to the returned channel, until
// the given context is done or the Connection is shutdown.
func PromptMonitor(ctx context.Context, client *Client, modes ...iterm2.PromptMonitorMode) (<-chan *iterm2.PromptNotification, error) {
	if len(modes) == 0 {
		modes = []iterm2.PromptMonitorMode{
			iterm2.PromptMonitorMode_COMMAND_START,
			iterm2.PromptMonitorMode_COMMAND_END,
			iterm2.PromptMonitorMode_PROMPT,
		}

	}
	req := NewNotificationRequest(true, iterm2.NotificationType_NOTIFY_ON_PROMPT, "")
	req.Arguments = &iterm2.NotificationRequest_PromptMonitorRequest{
		PromptMonitorRequest: &iterm2.PromptMonitorRequest{
			Modes: modes,
		},
	}

	notifications, err := client.Subscribe(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("prompt monitor: %w", err)
	}

	prompts := make(chan *iterm2.PromptNotification)

	go func() {
		for notification := range notifications {
			if notification.GetPromptNotification() != nil {
				prompts <- notification.GetPromptNotification()
			}
		}
		close(prompts)
	}()

	return prompts, nil
}
