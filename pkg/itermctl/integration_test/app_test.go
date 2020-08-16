// +build test_with_iterm

package integration_test

import (
	"fmt"
	appPkg "mrz.io/itermctl/pkg/itermctl/app"
	"mrz.io/itermctl/pkg/itermctl/rpc"
	"testing"
)

func TestApp_InvokeFunction(t *testing.T) {
	t.Parallel()

	client, conn, _ := createTestConn(t)
	defer conn.Close()
	app := appPkg.NewApp(client)

	argName := "foo"
	argValue := "bar"

	cb := rpc.NewRPC(
		"rpc_test_succeeding_func",
		func(args *rpc.Invocation) (interface{}, error) {
			return args.GetString(argName)
		},
		rpc.Arg{Name: argName},
	)

	rpc.Register(nil, client, cb)

	var result string
	err := app.InvokeFunction(fmt.Sprintf("rpc_test_succeeding_func(%s: %q)", argName, argValue), &result)
	if err != nil {
		t.Fatal(err)
	}

	if result != argValue {
		t.Fatalf("expected %q, got %q", argValue, result)
	}
}

func TestApp_InvokeFunction_WithError(t *testing.T) {
	t.Parallel()

	client, conn, _ := createTestConn(t)
	defer conn.Close()

	app := appPkg.NewApp(client)
	errorString := "error from the RPC function"

	cb := rpc.NewRPC(
		"rpc_test_failing_func",
		func(args *rpc.Invocation) (interface{}, error) {
			return nil, fmt.Errorf(errorString)
		},
	)

	if err := rpc.Register(nil, client, cb); err != nil {
		t.Fatal(err)
	}

	var result string
	err := app.InvokeFunction("rpc_test_failing_func()", &result)

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
	client, conn, _ := createTestConn(t)
	defer conn.Close()

	app := appPkg.NewApp(client)

	testWindowResp, err := app.CreateTab("", 0, "")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := app.CreateTab(testWindowResp.GetWindowId(), 0, "")
	if err != nil {
		t.Fatal(err)
	}

	err = app.CloseTab(true, resp.GetTabId())
	if err != nil {
		t.Fatal(err)
	}

	err = app.CloseWindow(true, resp.GetWindowId())
	if err != nil {
		t.Fatal(err)
	}
}
