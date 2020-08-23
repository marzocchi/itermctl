// +build test_with_iterm

package integration_test

import (
	"mrz.io/itermctl/pkg/itermctl"
	"mrz.io/itermctl/pkg/itermctl/internal/test"
	iterm2 "mrz.io/itermctl/pkg/itermctl/proto"
	"testing"
	"time"
)

func TestNewSessionMonitor(t *testing.T) {
	conn, err := itermctl.GetCredentialsAndConnect(test.AppName(t), true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := itermctl.NewClient(conn)

	app := itermctl.NewApp(client)

	newSessions, err := itermctl.MonitorNewSessions(nil, client)
	if err != nil {
		t.Fatal(err)
	}

	testWindowResp, err := app.CreateTab("", 0, "")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = app.CloseWindow(true, testWindowResp.GetWindowId())
		if err != nil {
			t.Fatal(err)
		}
	}()

	expectSessionId(t, newSessions, testWindowResp.GetSessionId())
}

func TestTerminateSessionMonitor(t *testing.T) {
	t.Parallel()

	conn, err := itermctl.GetCredentialsAndConnect(test.AppName(t), true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client := itermctl.NewClient(conn)
	closedSessions, err := itermctl.MonitorSessionsTermination(nil, client)
	if err != nil {
		t.Fatal(err)
	}

	app := itermctl.NewApp(client)

	testWindowResp, err := app.CreateTab("", 0, "")
	if err != nil {
		t.Fatal(err)
	}

	err = app.SendText(testWindowResp.GetSessionId(), "exit\n", false)
	if err != nil {
		t.Fatal(err)
	}

	expectSessionId(t, closedSessions, testWindowResp.GetSessionId())
}

func expectSessionId(t *testing.T, notifications interface{}, expectedSessionId string) {
	timeout := 1 * time.Second
	foundSessions := make(chan string)
	go func() {
		for {
			if ch, ok := notifications.(<-chan *iterm2.TerminateSessionNotification); ok {
				select {
				case notification := <-ch:
					if notification.GetSessionId() == expectedSessionId {
						foundSessions <- notification.GetSessionId()
						close(foundSessions)
					}
				}
			} else if ch, ok := notifications.(<-chan *iterm2.NewSessionNotification); ok {
				select {
				case notification := <-ch:
					if notification.GetSessionId() == expectedSessionId {
						foundSessions <- notification.GetSessionId()
						close(foundSessions)
					}
				}
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
