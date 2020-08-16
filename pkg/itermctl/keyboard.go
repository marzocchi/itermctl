package itermctl

import (
	"context"
	iterm2 "mrz.io/itermctl/pkg/itermctl/proto"
)

func MonitorKeystrokes(ctx context.Context, client *Client, sessionId string) (<-chan *iterm2.KeystrokeNotification, error) {
	if sessionId == "" {
		sessionId = AllSessions
	}

	var nt iterm2.NotificationType
	nt = iterm2.NotificationType_NOTIFY_ON_KEYSTROKE

	req := NewNotificationRequest(true, nt, "")
	notifications, err := client.Subscribe(ctx, req)

	if err != nil {
		return nil, err
	}

	keystrokes := make(chan *iterm2.KeystrokeNotification)

	go func() {
		for notification := range notifications {
			if notification.GetKeystrokeNotification() != nil {
				if sessionId != AllSessions && notification.GetKeystrokeNotification().GetSession() != sessionId {
					continue
				}

				keystrokes <- notification.GetKeystrokeNotification()
			}
		}
		close(keystrokes)
	}()

	return keystrokes, nil
}
