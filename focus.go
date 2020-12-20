package itermctl

import (
	"context"
	"mrz.io/itermctl/iterm2"
)

const (
	TabSelected WhichFocusUpdate = iota
	SessionSelected
	WindowBecameKey
	WindowIsCurrent
	WindowResignedKey
	ApplicationBecameActive
	ApplicationResignedActive
)

type WhichFocusUpdate int

func (w WhichFocusUpdate) String() string {
	return []string{"TabSelected", "SessionSelected", "WindowBecameKey", "WindowIsCurrent", "WindowResignedKey",
		"ApplicationBecameActive", "ApplicationResignedActive"}[w]
}

type FocusUpdate struct {
	Id    string
	Which WhichFocusUpdate
}

func MonitorFocus(ctx context.Context, conn *Connection) (<-chan FocusUpdate, error) {
	req := NewNotificationRequest(true, iterm2.NotificationType_NOTIFY_ON_FOCUS_CHANGE, "")
	recv, err := conn.Subscribe(ctx, req)

	if err != nil {
		return nil, err
	}

	updates := make(chan FocusUpdate)

	go func() {
		for n := range recv.Ch() {
			if n.GetNotification().GetFocusChangedNotification() != nil {
				update := n.GetNotification().GetFocusChangedNotification()
				updates <- GetFocusUpdate(update)
			}
		}

		close(updates)
	}()

	return updates, nil
}

func GetFocusUpdate(notification *iterm2.FocusChangedNotification) FocusUpdate {
	var update FocusUpdate

	switch notification.GetEvent().(type) {
	case *iterm2.FocusChangedNotification_SelectedTab:
		update = FocusUpdate{Id: notification.GetSelectedTab(), Which: TabSelected}

	case *iterm2.FocusChangedNotification_Session:
		update = FocusUpdate{Id: notification.GetSession(), Which: SessionSelected}

	case *iterm2.FocusChangedNotification_ApplicationActive:
		var which WhichFocusUpdate

		if notification.GetApplicationActive() {
			which = ApplicationBecameActive
		} else {
			which = ApplicationResignedActive
		}

		update = FocusUpdate{Which: which}

	case *iterm2.FocusChangedNotification_Window_:
		var which WhichFocusUpdate

		switch notification.GetWindow().GetWindowStatus() {
		case iterm2.FocusChangedNotification_Window_TERMINAL_WINDOW_BECAME_KEY:
			which = WindowBecameKey
		case iterm2.FocusChangedNotification_Window_TERMINAL_WINDOW_IS_CURRENT:
			which = WindowIsCurrent
		case iterm2.FocusChangedNotification_Window_TERMINAL_WINDOW_RESIGNED_KEY:
			which = WindowResignedKey
		}

		update = FocusUpdate{Id: notification.GetWindow().GetWindowId(), Which: which}
	}

	return update
}
