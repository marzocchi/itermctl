package integration_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"mrz.io/itermctl/pkg/itermctl"
	"regexp"
	"testing"
	"time"
)

func TestCustomControlSequenceMonitor(t *testing.T) {
	identity := "foo"
	escaper := itermctl.NewCustomControlSequenceEscaper(identity)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	re := regexp.MustCompile("test-sequence")
	notifications, err := conn.MonitorCustomControlSequences(ctx, identity, re, itermctl.AllSessions)
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

	sessionId := testWindowResp.GetSessionId()
	session, err := app.Session(sessionId)
	if err != nil {
		t.Fatal(err)
	}

	tempFile, err := ioutil.TempFile("", "*-custom_control_test")
	if err != nil {
		t.Fatal(err)
	}

	_, err = tempFile.Write([]byte(escaper.Escape("test-sequence")))
	if err != nil {
		t.Fatal(err)
	}

	if err := session.SendText(fmt.Sprintf("cat %s\n", tempFile.Name()), false); err != nil {
		t.Fatal(err)
	}

	select {
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	case notification := <-notifications:
		if testWindowResp.GetSessionId() != notification.Notification.GetSession() {
			t.Fatalf("expected %q, got %q", testWindowResp.GetSessionId(), notification.Notification.GetSession())
		}
		if notification.Matches[0] != "test-sequence" {
			t.Fatalf("expected %q, got %q", "test-sequence", notification.Matches[0])
		}
	}
}
