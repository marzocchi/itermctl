package itermctl

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"mrz.io/itermctl/pkg/itermctl/proto"
	"regexp"
)

type Escaper struct {
	identity string
}

// NewCustomEscaper creates an Escaper bound to the given identity.
func NewCustomEscaper(identity string) *Escaper {
	return &Escaper{identity: identity}
}

// Escape wraps a format string in a Custom Control Sequence
func (e *Escaper) Escape(format string, a ...interface{}) string {
	return fmt.Sprintf("\033]1337;Custom=id=%s:%s\a",
		e.identity, fmt.Sprintf(format, a...))
}

type CustomControlSequenceNotification struct {
	Matches      []string
	Notification *iterm2.CustomEscapeSequenceNotification
}

// MonitorCustomControlSequences subscribes to Custom Control Sequence Notifications and writes every Notification that
// matches any of the given sessionId, identities and regex to the returned channel, until the given context is done or
// the Connection is closed.
// An identity is a secret shared between the client and iTerm2 and is required as a security mechanism. Note that
// filtering against unknown identities is done here, in the client side, before writing a Notification to the channel.
// See https://www.iterm2.com/python-api/customcontrol.html.
func MonitorCustomControlSequences(ctx context.Context, client *Client, identity string, re *regexp.Regexp, sessionId string) (<-chan CustomControlSequenceNotification, error) {
	notifications := make(chan CustomControlSequenceNotification)

	req := NewNotificationRequest(true, iterm2.NotificationType_NOTIFY_ON_CUSTOM_ESCAPE_SEQUENCE, sessionId)
	recv, err := client.Subscribe(ctx, req)

	if err != nil {
		return nil, fmt.Errorf("custom control sequence monitor: %w", err)
	}

	go func() {
		for notification := range recv {
			ccsNotification := notification.GetCustomEscapeSequenceNotification()
			if ccsNotification == nil {
				continue
			}

			if ccsNotification.GetSenderIdentity() != identity {
				log.Warnf(
					"custom control sequence monitor: ignoring notification as sender identity %q does not match expected %q",
					ccsNotification.GetSenderIdentity(),
					identity,
				)
				continue
			}

			matches := re.FindStringSubmatch(ccsNotification.GetPayload())
			if len(matches) < 1 {
				continue
			}

			notifications <- CustomControlSequenceNotification{
				Notification: ccsNotification,
				Matches:      matches,
			}
		}
		close(notifications)
	}()

	return notifications, nil
}
