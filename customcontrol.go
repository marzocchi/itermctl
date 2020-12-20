package itermctl

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"mrz.io/itermctl/iterm2"
	"regexp"
)

type CustomControlSequenceEscaper struct {
	identity string
}

// NewCustomControlSequenceEscaper creates an CustomControlSequenceEscaper bound to the given identity.
func NewCustomControlSequenceEscaper(identity string) *CustomControlSequenceEscaper {
	return &CustomControlSequenceEscaper{identity: identity}
}

// Escape wraps a formatted string in a Custom Control Sequence
func (e *CustomControlSequenceEscaper) Escape(format string, a ...interface{}) string {
	return fmt.Sprintf("\033]1337;Custom=id=%s:%s\a",
		e.identity, fmt.Sprintf(format, a...))
}

type CustomControlSequenceNotification struct {
	Matches      []string
	Notification *iterm2.CustomEscapeSequenceNotification
}

// MonitorCustomControlSequences subscribes to CustomControlSequenceNotification and writes each one that matches any
// of the given sessionId, identities and regex to the returned channel, until the context is done or the Connection is
// closed. An identity is a secret shared between the conn and iTerm2 and is required as a security mechanism. Note that
// filtering against unknown identities is done here on the client side.
// See https://www.iterm2.com/python-api/customcontrol.html.
func MonitorCustomControlSequences(ctx context.Context, conn *Connection, identity string, re *regexp.Regexp, sessionId string) (<-chan CustomControlSequenceNotification, error) {
	notifications := make(chan CustomControlSequenceNotification)

	req := NewNotificationRequest(true, iterm2.NotificationType_NOTIFY_ON_CUSTOM_ESCAPE_SEQUENCE, sessionId)
	recv, err := conn.Subscribe(ctx, req)

	if err != nil {
		return nil, fmt.Errorf("custom control sequence monitor: %w", err)
	}

	go func() {
		for msg := range recv.Ch() {
			notification := msg.GetNotification().GetCustomEscapeSequenceNotification()
			if notification == nil {
				continue
			}

			if notification.GetSenderIdentity() != identity {
				logrus.Warnf(
					"custom control sequence monitor: ignoring msg as sender identity %q does not match expected %q",
					notification.GetSenderIdentity(),
					identity,
				)
				continue
			}

			matches := re.FindStringSubmatch(notification.GetPayload())
			if len(matches) < 1 {
				continue
			}

			notifications <- CustomControlSequenceNotification{
				Notification: notification,
				Matches:      matches,
			}
		}
		close(notifications)
	}()

	return notifications, nil
}
