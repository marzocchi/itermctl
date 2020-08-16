// +build test_with_iterm

package integration_test

import (
	"mrz.io/itermctl/pkg/itermctl"
	appPkg "mrz.io/itermctl/pkg/itermctl/app"
	"testing"
	"time"
)

func TestNewSessionMonitor(t *testing.T) {
	client, conn, _ := createTestConn(t)
	defer conn.Close()

	app := appPkg.NewApp(client)

	newSessions, err := itermctl.NewSessionMonitor(nil, client)
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

	client, conn, _ := createTestConn(t)
	defer conn.Close()

	closedSessions, err := itermctl.TerminateSessionMonitor(nil, client)
	if err != nil {
		t.Fatal(err)
	}

	app := appPkg.NewApp(client)

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

func expectSessionId(t *testing.T, closedSessions <-chan itermctl.SessionId, expectedSessionId string) {
	timeout := 1 * time.Second
	foundSessions := make(chan string)
	go func() {
		for {
			select {
			case sessionId := <-closedSessions:
				if sessionId == expectedSessionId {
					foundSessions <- sessionId
					close(foundSessions)
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
