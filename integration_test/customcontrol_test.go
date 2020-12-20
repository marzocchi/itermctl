// +build test_with_iterm

package integration_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"mrz.io/itermctl"
	"mrz.io/itermctl/internal/test"
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
	notifications, err := itermctl.MonitorCustomControlSequences(ctx, conn, identity, re, itermctl.AllSessions)
	if err != nil {
		t.Fatal(err)
	}

	testWindowResp, closeTestWindow := test.CreateWindow(app, t)
	defer closeTestWindow()

	sessionId := testWindowResp.GetSessionId()
	session := app.Session(sessionId)
	if session == nil {
		t.Fatal("no session")
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
