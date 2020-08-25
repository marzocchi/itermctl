package itermctl

import (
	"context"
	iterm2 "mrz.io/itermctl/pkg/itermctl/proto"
)

// MonitorKeystrokes subscribes to KeystrokeNotification and writes each one to the returned channel, until the context
// is canceled or the Connection is closed.
func (conn *Connection) MonitorKeystrokes(ctx context.Context, sessionId string) (<-chan *iterm2.KeystrokeNotification, error) {
	if sessionId == "" {
		sessionId = AllSessions
	}

	var nt iterm2.NotificationType
	nt = iterm2.NotificationType_NOTIFY_ON_KEYSTROKE

	req := NewNotificationRequest(true, nt, "")
	recv, err := conn.Subscribe(ctx, req)

	if err != nil {
		return nil, err
	}

	keystrokes := make(chan *iterm2.KeystrokeNotification)

	go func() {
		for msg := range recv.Ch() {
			if msg.GetNotification().GetKeystrokeNotification() != nil {
				if sessionId != AllSessions && msg.GetNotification().GetKeystrokeNotification().GetSession() != sessionId {
					continue
				}

				keystrokes <- msg.GetNotification().GetKeystrokeNotification()
			}
		}
		close(keystrokes)
	}()

	return keystrokes, nil
}
