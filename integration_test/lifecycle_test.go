// +build test_with_iterm

package integration_test

import (
	"context"
	"mrz.io/itermctl"
	"mrz.io/itermctl/internal/test"
	"mrz.io/itermctl/iterm2"
	"testing"
	"time"
)

func TestNewSessionMonitor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	newSessions, err := itermctl.MonitorNewSessions(ctx, conn)
	if err != nil {
		t.Fatal(err)
	}

	testWindowResp, closeTestWindow := test.CreateWindow(app, t)
	defer closeTestWindow()

	expectSessionNotification(newSessions, testWindowResp.GetSessionId(), t)
}

func TestTerminateSessionMonitor(t *testing.T) {
	defer test.AssertNoLeftoverWindows(app, t)
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	closedSessions, err := itermctl.MonitorSessionsTermination(ctx, conn)
	if err != nil {
		t.Fatal(err)
	}

	testWindowResp, _ := test.CreateWindow(app, t)

	session := app.Session(testWindowResp.GetSessionId())
	if session == nil {
		t.Fatalf("no session: %s", testWindowResp.GetSessionId())
	}

	err = session.SendText("exit\n", false)
	if err != nil {
		t.Fatal(err)
	}

	expectSessionNotification(closedSessions, testWindowResp.GetSessionId(), t)
}

func expectSessionNotification(notifications interface{}, expectedSessionId string, t *testing.T) {
	timeout := 5 * time.Second
	foundSessions := make(chan string)

	go func() {
		for {
			if ch, ok := notifications.(<-chan *iterm2.TerminateSessionNotification); ok {
				for notification := range ch {
					if notification.GetSessionId() == expectedSessionId {
						foundSessions <- notification.GetSessionId()
						close(foundSessions)
					}
				}
			} else if ch, ok := notifications.(<-chan *iterm2.NewSessionNotification); ok {
				for notification := range ch {
					if notification.GetSessionId() == expectedSessionId {
						foundSessions <- notification.GetSessionId()
						close(foundSessions)
					}
				}
			} else {
				t.Fatal("expected <- chan of *iterm2.TerminateSessionNotification or *iterm2.NewSessionNotification")
			}
		}
	}()

	select {
	case <-time.After(timeout):
		t.Fatalf("timeout of %s exceeded while waiting for expected session ID %q", timeout, expectedSessionId)
	case foundSession := <-foundSessions:
		if foundSession == "" {
			t.Fatalf("empty session ID received")
		}
	}
}
