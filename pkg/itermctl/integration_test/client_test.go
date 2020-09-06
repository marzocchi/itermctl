// +build test_with_iterm

package integration_test

import (
	"context"
	"fmt"
	"mrz.io/itermctl/pkg/itermctl"
	iterm2 "mrz.io/itermctl/pkg/itermctl/proto"
	"os"
	"testing"
	"time"
)

var conn *itermctl.Connection
var app *itermctl.App

const profileName = "itermctl test profile"


func TestMain(m *testing.M) {
	var err error

	itermctl.WaitResponseTimeout = 20 * time.Second
	conn, err = itermctl.GetCredentialsAndConnect("itermctl_integration_test", true)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	app, err = itermctl.NewApp(conn)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	os.Exit(func() int {
		defer conn.Close()
		return m.Run()
	}())
}

func TestClient_CloseConnectionDuringGetResponse(t *testing.T) {
	conn, err := itermctl.GetCredentialsAndConnect("itermctl_integration_test", true)
	if err != nil {
		t.Fatal(err)
	}

	funcName := "itermctl_test_sleep_func"

	err = conn.RegisterRpc(context.Background(), itermctl.Rpc{
		Name: funcName,
		Args: nil,
		F: func(args *itermctl.RpcInvocation) (interface{}, error) {
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

	window1Resp, err := app.CreateTab("", 0, profileName)
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
	case n := <-recv.Ch():
		if n.GetNotification().GetNewSessionNotification().GetSessionId() != window1Resp.GetSessionId() {
			t.Fatalf("expected notification for session %s, got %s", window1Resp.GetSessionId(),
				n.GetNotification().GetNewSessionNotification().GetSessionId())
		}
	}

	cancel()

	window2Resp, err := app.CreateTab("", 0, profileName)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = app.CloseWindow(true, window2Resp.GetWindowId())
		if err != nil {
			t.Fatal(err)
		}
	}()

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
