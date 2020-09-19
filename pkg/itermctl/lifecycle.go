package itermctl

import (
	"context"
	"mrz.io/itermctl/pkg/itermctl/iterm2"
)

// MonitorNewSessions subscribes to NewSessionNotifications and forwards each one to the returned channel, until the
// given context is done or the Connection is shutdown.
func MonitorNewSessions(ctx context.Context, conn *Connection) (<-chan *iterm2.NewSessionNotification, error) {
	req := NewNotificationRequest(true, iterm2.NotificationType_NOTIFY_ON_NEW_SESSION, "")
	recv, err := conn.Subscribe(ctx, req)

	if err != nil {
		return nil, err
	}

	notifications := make(chan *iterm2.NewSessionNotification)

	go func() {
		for n := range recv.Ch() {
			if n.GetNotification().GetNewSessionNotification() != nil {
				notifications <- n.GetNotification().GetNewSessionNotification()
			}
		}

		close(notifications)
	}()

	return notifications, nil
}

// MonitorSessionsTermination subscribes to TerminateSessionNotification and writes each one to the returned channel,
// until the given context is done or the Connection is shutdown.
func MonitorSessionsTermination(ctx context.Context, conn *Connection) (<-chan *iterm2.TerminateSessionNotification, error) {
	req := NewNotificationRequest(true, iterm2.NotificationType_NOTIFY_ON_TERMINATE_SESSION, "")
	recv, err := conn.Subscribe(ctx, req)

	if err != nil {
		return nil, err
	}

	notifications := make(chan *iterm2.TerminateSessionNotification)

	go func() {
		for n := range recv.Ch() {
			if n.GetNotification().GetTerminateSessionNotification() != nil {
				notifications <- n.GetNotification().GetTerminateSessionNotification()
			}
		}

		close(notifications)
	}()

	return notifications, nil
}
