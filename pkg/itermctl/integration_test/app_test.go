// +build test_with_iterm

package integration_test

import (
	"context"
	"fmt"
	"mrz.io/itermctl/pkg/itermctl"
	"mrz.io/itermctl/pkg/itermctl/internal/test"
	"testing"
	"time"
)

func TestConnection_InvokeFunction(t *testing.T) {
	type A struct {
		Foo string
	}

	args := A{Foo: "bar"}

	rpc := itermctl.Rpc{
		Name: "rpc_test_succeeding_func",
		Args: args,
		F: func(invocation *itermctl.RpcInvocation) (interface{}, error) {
			invocation.Args(&args)
			return args.Foo, nil
		},
	}

	conn.RegisterRpc(context.Background(), rpc)

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

	errorString := "error from the RpcFunc function"

	rpc := itermctl.Rpc{
		Name: "rpc_test_failing_func",
		Args: nil,
		F: func(args *itermctl.RpcInvocation) (interface{}, error) {
			return nil, fmt.Errorf(errorString)
		},
	}

	if err := conn.RegisterRpc(context.Background(), rpc); err != nil {
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
	testWindowResp, err := app.CreateTab("", 0, "")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := app.CreateTab(testWindowResp.GetWindowId(), 0, profileName)
	if err != nil {
		t.Fatal(err)
	}

	err = app.CloseTab(true, fmt.Sprintf("%d", resp.GetTabId()))
	if err != nil {
		t.Fatal(err)
	}

	err = app.CloseWindow(true, resp.GetWindowId())
	if err != nil {
		t.Fatal(err)
	}
}
