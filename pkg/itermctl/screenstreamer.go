package itermctl

import (
	"context"
	iterm2 "mrz.io/itermctl/pkg/itermctl/proto"
)

// MonitorScreenUpdates subscribes to ScreenUpdateNotification and forwards each one to the returned channel.
// Subscription lasts until the given context is canceled or the client's connection is closed. Use methods such as
// App.ScreenContents to retrieve the screen's contents.
func MonitorScreenUpdates(ctx context.Context, client *Client, sessionId string) (<-chan *iterm2.ScreenUpdateNotification, error) {
	notifications := make(chan *iterm2.ScreenUpdateNotification)

	req := NewNotificationRequest(true, iterm2.NotificationType_NOTIFY_ON_SCREEN_UPDATE, sessionId)
	src, err := client.Subscribe(ctx, req)

	if err != nil {
		return nil, err
	}

	go func() {
		for n := range src {
			if n.GetScreenUpdateNotification() != nil {
				notifications <- n.GetScreenUpdateNotification()
			}
		}
	}()

	return notifications, nil
}
