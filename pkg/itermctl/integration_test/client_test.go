// +build test_with_iterm

package integration_test

import (
	"context"
	"fmt"
	"mrz.io/itermctl/pkg/itermctl"
	"mrz.io/itermctl/pkg/itermctl/internal/seq"
	"mrz.io/itermctl/pkg/itermctl/internal/test"
	iterm2 "mrz.io/itermctl/pkg/itermctl/proto"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	err := itermctl.AuthDisabled()
	if err != nil {
		fmt.Fprintf(os.Stderr, "auth must be disabled to run these tests\n")
		fmt.Fprintf(os.Stderr, "error is: %s\n", err)
		os.Exit(1)
	}

	itermctl.WaitResponseTimeout = 15 * time.Second

	ctx, cancel := context.WithCancel(context.Background())
	test.StartTakingScreenshots(ctx, "/Users/runner/work/itermctl/itermctl/artifacts")

	code := m.Run()
	cancel()

	os.Exit(code)
}

func TestClient_CloseConnectionDuringGetResponse(t *testing.T) {
	conn, err := itermctl.GetCredentialsAndConnect(test.AppName(t), true)
	if err != nil {
		t.Fatal(err)
	}

	client := itermctl.NewClient(conn)

	funcName := "itermctl_test_sleep_func"

	itermctl.RegisterRpc(context.Background(), client, itermctl.Rpc{
		Name: funcName,
		Args: nil,
		F: func(args *itermctl.RpcInvocation) (interface{}, error) {
			<-time.After(1 * time.Second)
			return nil, nil
		},
	})

	invocation := fmt.Sprintf("%s()", funcName)

	req := &iterm2.ClientOriginatedMessage{
		Id: seq.MessageId.Next(),
		Submessage: &iterm2.ClientOriginatedMessage_InvokeFunctionRequest{
			InvokeFunctionRequest: &iterm2.InvokeFunctionRequest{
				Context:    &iterm2.InvokeFunctionRequest_App_{},
				Invocation: &invocation,
			},
		},
	}

	respCh, err := client.Request(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		<-time.After(500 * time.Millisecond)
		conn.Close()
	}()

	resp := <-respCh
	if resp != nil {
		t.Fatalf("expected resp = nil, got %s", resp)
	}
}

func TestClient_Subscribe(t *testing.T) {
	conn, err := itermctl.GetCredentialsAndConnect(test.AppName(t), true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := itermctl.NewClient(conn)

	app := itermctl.NewApp(client)

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
