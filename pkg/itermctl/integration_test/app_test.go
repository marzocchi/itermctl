// +build test_with_iterm

package integration_test

import (
	"context"
	"fmt"
	"mrz.io/itermctl/pkg/itermctl"
	"mrz.io/itermctl/pkg/itermctl/internal/test"
	"testing"
)

func TestApp_InvokeFunction(t *testing.T) {
	t.Parallel()

	conn, err := itermctl.GetCredentialsAndConnect(test.AppName(t), true)
	defer conn.Close()
	client := itermctl.NewClient(conn)
	app := itermctl.NewApp(client)

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

	itermctl.RegisterRpc(context.TODO(), client, rpc)

	var result string
	err = app.InvokeFunction(fmt.Sprintf("rpc_test_succeeding_func(%s: %q)", "foo", args.Foo), &result)
	if err != nil {
		t.Fatal(err)
	}

	if result != args.Foo {
		t.Fatalf("expected %q, got %q", args.Foo, result)
	}
}

func TestApp_InvokeFunction_WithError(t *testing.T) {
	t.Parallel()

	conn, err := itermctl.GetCredentialsAndConnect(test.AppName(t), true)
	defer conn.Close()
	client := itermctl.NewClient(conn)

	app := itermctl.NewApp(client)
	errorString := "error from the RpcFunc function"

	rpc := itermctl.Rpc{
		Name: "rpc_test_failing_func",
		Args: nil,
		F: func(args *itermctl.RpcInvocation) (interface{}, error) {
			return nil, fmt.Errorf(errorString)
		},
	}

	if err := itermctl.RegisterRpc(nil, client, rpc); err != nil {
		t.Fatal(err)
	}

	var result string
	err = app.InvokeFunction("rpc_test_failing_func()", &result)

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
	conn, err := itermctl.GetCredentialsAndConnect(test.AppName(t), true)
	defer conn.Close()
	client := itermctl.NewClient(conn)

	app := itermctl.NewApp(client)

	testWindowResp, err := app.CreateTab("", 0, "")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := app.CreateTab(testWindowResp.GetWindowId(), 0, "")
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
