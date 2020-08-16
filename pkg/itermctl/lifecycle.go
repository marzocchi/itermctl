package itermctl

import (
	"context"
	iterm2 "mrz.io/itermctl/pkg/itermctl/proto"
)

type SessionId = string

// MonitorNewSessions subscribes to NewSessionNotifications and forwards each one to the returned channel, until the
// given context is done or the Connection is shutdown.
func MonitorNewSessions(ctx context.Context, client *Client) (<-chan *iterm2.NewSessionNotification, error) {
	req := NewNotificationRequest(true, iterm2.NotificationType_NOTIFY_ON_NEW_SESSION, "")
	src, err := client.Subscribe(ctx, req)

	if err != nil {
		return nil, err
	}

	notifications := make(chan *iterm2.NewSessionNotification)

	go func() {
		for n := range src {
			if n.GetNewSessionNotification() != nil {
				notifications <- n.GetNewSessionNotification()
			}
		}

		close(notifications)
	}()

	return notifications, nil
}

// MonitorSessionsTermination subscribes to session termination notifications and writes the closed session's IDs to the
// channel, until the given context is done or the Connection is shutdown.
func MonitorSessionsTermination(ctx context.Context, client *Client) (<-chan *iterm2.TerminateSessionNotification, error) {
	req := NewNotificationRequest(true, iterm2.NotificationType_NOTIFY_ON_TERMINATE_SESSION, "")
	src, err := client.Subscribe(ctx, req)

	if err != nil {
		return nil, err
	}

	notifications := make(chan *iterm2.TerminateSessionNotification)

	go func() {
		for n := range src {
			if n.GetTerminateSessionNotification() != nil {
				notifications <- n.GetTerminateSessionNotification()
			}
		}

		close(notifications)
	}()

	return notifications, nil
}
