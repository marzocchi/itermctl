package itermctl

import (
	"context"
	iterm2 "mrz.io/itermctl/pkg/itermctl/proto"
)

// MonitorNewSessions subscribes to NewSessionNotifications and forwards each one to the returned channel, until the
// given context is done or the Connection is shutdown.
func (conn *Connection) MonitorNewSessions(ctx context.Context) (<-chan *iterm2.NewSessionNotification, error) {
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
func (conn *Connection) MonitorSessionsTermination(ctx context.Context) (<-chan *iterm2.TerminateSessionNotification, error) {
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
