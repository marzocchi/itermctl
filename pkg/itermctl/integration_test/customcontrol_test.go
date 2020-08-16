// +build test_with_iterm

package integration_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"mrz.io/itermctl/pkg/itermctl"
	appPkg "mrz.io/itermctl/pkg/itermctl/app"
	"regexp"
	"testing"
	"time"
)

func TestCustomControlSequenceMonitor(t *testing.T) {
	identity := "foo"

	escaper := itermctl.NewCustomEscaper(identity)

	client, conn, _ := createTestConn(t)
	defer conn.Close()
	app := appPkg.NewApp(client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	re := regexp.MustCompile("test-sequence")
	notifications, err := itermctl.CustomControlSequenceMonitor(ctx, client, identity, re, itermctl.AllSessions)
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

	tempFile, err := ioutil.TempFile("", "*-custom_control_test")
	if err != nil {
		t.Fatal(err)
	}

	_, err = tempFile.Write([]byte(escaper.Escape("test-sequence")))
	if err != nil {
		t.Fatal(err)
	}

	if err := app.SendText(sessionId, fmt.Sprintf("cat %s\n", tempFile.Name()), false); err != nil {
		t.Fatal(err)
	}

	select {
	case <-time.After(1 * time.Second):
		t.Fatal("timed out")
	case notification := <-notifications:
		if testWindowResp.GetSessionId() != notification.Message.GetSession() {
			t.Fatalf("expected %q, got %q", testWindowResp.GetSessionId(), notification.Message.GetSession())
		}
		if notification.Matches[0] != "test-sequence" {
			t.Fatalf("expected %q, got %q", "test-sequence", notification.Matches[0])
		}
	}
}
