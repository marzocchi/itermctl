package itermctl

import (
	"context"
	iterm2 "mrz.io/itermctl/pkg/itermctl/proto"
)

type SessionId = string

// NewSessionMonitor subscribes to new session notifications and writes new session's IDs to the channel, until the
// given context is done or the Connection is shutdown.
func NewSessionMonitor(ctx context.Context, client *Client) (<-chan SessionId, error) {
	return sessionLifecycleMonitor(ctx, client, true)
}

// TerminateSessionMonitor subscribes to session termination notifications and writes the closed session's IDs to the
// channel, until the given context is done or the Connection is shutdown.
func TerminateSessionMonitor(ctx context.Context, conn *Client) (<-chan SessionId, error) {
	return sessionLifecycleMonitor(ctx, conn, false)
}

type sessionNotification interface {
	GetSessionId() string
}

func sessionLifecycleMonitor(ctx context.Context, conn *Client, expectNewSession bool) (<-chan SessionId, error) {
	var nt iterm2.NotificationType
	if expectNewSession {
		nt = iterm2.NotificationType_NOTIFY_ON_NEW_SESSION
	} else {
		nt = iterm2.NotificationType_NOTIFY_ON_TERMINATE_SESSION
	}

	req := NewNotificationRequest(true, nt, "")
	notifications, err := conn.Subscribe(ctx, req)

	if err != nil {
		return nil, err
	}

	sessionIds := make(chan SessionId)

	go func() {
		for notification := range notifications {
			var sn sessionNotification

			if expectNewSession {
				sn = notification.GetNewSessionNotification()
			} else {
				sn = notification.GetTerminateSessionNotification()
			}

			if sn.GetSessionId() == "" {
				continue
			}

			sessionIds <- sn.GetSessionId()
		}
		close(sessionIds)
	}()

	return sessionIds, nil
}
