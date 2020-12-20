// +build test_with_iterm

package integration_test

import (
	"context"
	"fmt"
	"mrz.io/itermctl"
	"mrz.io/itermctl/internal/test"
	"mrz.io/itermctl/iterm2"
	"mrz.io/itermctl/rpc"
	"testing"
	"time"
)

func TestClient_CloseConnectionDuringGetResponse(t *testing.T) {
	conn, err := itermctl.GetCredentialsAndConnect("itermctl_integration_test", true)
	if err != nil {
		t.Fatal(err)
	}

	funcName := "itermctl_test_sleep_func"

	err = rpc.Register(context.Background(), conn, rpc.RPC{
		Name: funcName,
		Args: nil,
		Function: func(args *rpc.Invocation) (interface{}, error) {
			<-time.After(1 * time.Second)
			return nil, nil
		},
	})

	if err != nil {
		t.Fatal(err)
	}

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
	// TODO change assertion on expected sessions so that it can be run in parallel
	// t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	req := itermctl.NewNotificationRequest(true, iterm2.NotificationType_NOTIFY_ON_NEW_SESSION, "")
	recv, err := conn.Subscribe(ctx, req)

	if err != nil {
		t.Fatal(err)
	}

	window1Resp, closeWindow1 := test.CreateWindow(app, t)
	defer closeWindow1()

	select {
	case <-time.After(1 * time.Second):
		t.Fatalf("timed out while waiting for first notification")
	case n := <-recv.Ch():
		if n.GetNotification().GetNewSessionNotification().GetSessionId() != window1Resp.GetSessionId() {
			t.Fatalf("expected notification for session %s, got %s", window1Resp.GetSessionId(),
				n.GetNotification().GetNewSessionNotification().GetSessionId())
		}
	}

	cancel()

	_, closeWindow2 := test.CreateWindow(app, t)
	defer closeWindow2()

	var notificationsAfterCancel []*iterm2.ServerOriginatedMessage

	select {
	case <-time.After(1 * time.Second):
		t.Fatalf("timed out while waiting for channel to close")
	case n, ok := <-recv.Ch():
		if !ok {
			break
		}
		notificationsAfterCancel = append(notificationsAfterCancel, n)
	}

	if len(notificationsAfterCancel) > 0 {
		t.Fatalf("expected no notification after context cancel, got %d", len(notificationsAfterCancel))
	}
}
