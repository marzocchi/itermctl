// +build test_with_iterm

package integration_test

import (
	"context"
	"fmt"
	"mrz.io/itermctl/pkg/itermctl"
	"mrz.io/itermctl/pkg/itermctl/internal/test"
	iterm2 "mrz.io/itermctl/pkg/itermctl/proto"
	"testing"
	"time"
)

func TestClient_CloseConnectionDuringGetResponse(t *testing.T) {
	conn, err := itermctl.GetCredentialsAndConnect(test.AppName(t), true)

	funcName := "itermctl_test_sleep_func"

	conn.RegisterRpc(context.Background(), itermctl.Rpc{
		Name: funcName,
		Args: nil,
		F: func(args *itermctl.RpcInvocation) (interface{}, error) {
			<-time.After(1 * time.Second)
			return nil, nil
		},
	})

	invocation := fmt.Sprintf("%s()", funcName)

	req := &iterm2.ClientOriginatedMessage{
		Submessage: &iterm2.ClientOriginatedMessage_InvokeFunctionRequest{
			InvokeFunctionRequest: &iterm2.InvokeFunctionRequest{
				Context:    &iterm2.InvokeFunctionRequest_App_{},
				Invocation: &invocation,
			},
		},
	}

	go func() {
		<-time.After(500 * time.Millisecond)
		conn.Close()
	}()

	resp, err := conn.GetResponse(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if resp != nil {
		t.Fatalf("expected resp = nil, got %s", resp)
	}
}

func TestClient_Subscribe(t *testing.T) {
	conn, err := itermctl.GetCredentialsAndConnect(test.AppName(t), true)
	defer func() {
		conn.Close()
	}()

	app, err := itermctl.NewApp(conn)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	req := itermctl.NewNotificationRequest(true, iterm2.NotificationType_NOTIFY_ON_NEW_SESSION, "")
	recv, err := conn.Subscribe(ctx, req)

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
	case n1 := <-recv.Ch():
		if n1.GetNotification().GetNewSessionNotification().GetSessionId() != window1Resp.GetSessionId() {
			t.Fatalf("expected %q, got %q", window1Resp.GetSessionId(), n1.GetNotification().GetNewSessionNotification().GetSessionId())
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
	case _, ok := <-recv.Ch():
		if ok != false {
			t.Fatal("expected channel to be closed")
		}
	}
}
