// +build test_with_iterm

package integration_test

import (
	"context"
	"fmt"
	"mrz.io/itermctl"
	"mrz.io/itermctl/internal/test"
	"mrz.io/itermctl/rpc"
	"testing"
	"time"
)

func TestConnection_InvokeFunction(t *testing.T) {
	type A struct {
		Foo string
	}

	args := A{Foo: "bar"}

	if err := rpc.Register(context.Background(), conn, rpc.RPC{
		Name: "rpc_test_succeeding_func",
		Args: args,
		Function: func(invocation *rpc.Invocation) (interface{}, error) {
			_ = invocation.Args(&args)
			return args.Foo, nil
		},
	}); err != nil {
		t.Fatal(err)
	}

	var result string
	err := conn.InvokeFunction(fmt.Sprintf("rpc_test_succeeding_func(%s: %q)", "foo", args.Foo), &result)
	if err != nil {
		t.Fatal(err)
	}

	if result != args.Foo {
		t.Fatalf("expected %q, got %q", args.Foo, result)
	}
}

func TestConnection_InvokeFunction_WithError(t *testing.T) {
	conn, err := itermctl.GetCredentialsAndConnect(test.AppName(t), true)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		conn.Close()
		<-time.After(1 * time.Second)
	}()

	errorString := "error from the RPC function"

	if err := rpc.Register(context.Background(), conn, rpc.RPC{
		Name: "rpc_test_failing_func",
		Args: nil,
		Function: func(args *rpc.Invocation) (interface{}, error) {
			return nil, fmt.Errorf(errorString)
		},
	}); err != nil {
		t.Fatal(err)
	}

	var result string
	err = conn.InvokeFunction("rpc_test_failing_func()", &result)

	if err == nil {
		t.Fatal(err)
	}

	expectedErrorMessage := fmt.Sprintf("FAILED: %s", errorString)
	if err.Error() != expectedErrorMessage {
		t.Fatalf("expected %q, got %q", expectedErrorMessage, err)
	}

	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}
}

func TestApp_CreateTab_CloseTab(t *testing.T) {
	testWindowResp, closeTestWindow := test.CreateWindow(app, t)
	defer closeTestWindow()

	resp, err := app.CreateTab(testWindowResp.GetWindowId(), 0, profileName)
	if err != nil {
		t.Fatal(err)
	}

	err = app.CloseTab(true, fmt.Sprintf("%d", resp.GetTabId()))
	if err != nil {
		t.Fatal(err)
	}
}

func TestApp_StateTracking(t *testing.T) {
	app.ActiveSession()
}
