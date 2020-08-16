// +build test_with_iterm

package integration_test

import (
	"context"
	"mrz.io/itermctl/pkg/itermctl"
	appPkg "mrz.io/itermctl/pkg/itermctl/app"
	iterm2 "mrz.io/itermctl/pkg/itermctl/proto"
	"testing"
	"time"
)

func TestClient_Subscribe(t *testing.T) {
	client, conn, _ := createTestConn(t)
	defer conn.Close()

	app := appPkg.NewApp(client)

	ctx, cancel := context.WithCancel(context.Background())
	req := itermctl.NewNotificationRequest(true, iterm2.NotificationType_NOTIFY_ON_NEW_SESSION, "")
	notifications, err := client.Subscribe(ctx, req)

	if err != nil {
		t.Fatal(err)
	}

	window1Resp, err := app.CreateTab("", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = app.CloseWindow(true, window1Resp.GetWindowId())
		if err != nil {
			t.Fatal(err)
		}
	}()

	select {
	case <-time.After(1 * time.Second):
		t.Fatalf("timed out while waiting for first notification")
	case n1 := <-notifications:
		if n1.GetNewSessionNotification().GetSessionId() != window1Resp.GetSessionId() {
			t.Fatalf("expected %q, got %q", window1Resp.GetSessionId(), n1.GetNewSessionNotification().GetSessionId())
		}
	}

	cancel()

	window2Resp, err := app.CreateTab("", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = app.CloseWindow(true, window2Resp.GetWindowId())
		if err != nil {
			t.Fatal(err)
		}
	}()

	select {
	case <-time.After(1 * time.Second):
		t.Fatalf("timed out while waiting for first notification")
	case _, ok := <-notifications:
		if ok != false {
			t.Fatal("expected channel to be closed")
		}
	}
}
