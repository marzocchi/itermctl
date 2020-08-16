// +build test_with_iterm

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"mrz.io/itermctl/pkg/itermctl"
	appPkg "mrz.io/itermctl/pkg/itermctl/app"
	"mrz.io/itermctl/pkg/itermctl/rpc"
	"testing"
)

func createTestConn(t *testing.T) (*itermctl.Client, itermctl.Connection, itermctl.Credentials) {
	creds, err := itermctl.GetCookieAndKey("itermctl test", true)

	if err != nil {
		t.Fatalf("could not get cookie: %s", err)
	}

	conn, err := itermctl.Connect(creds)
	if err != nil {
		t.Fatalf("could not connect to iTerm2: %s", err)
	}

	client := itermctl.NewClient(conn)

	return client, conn, creds
}

func TestRegisterCallback(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, conn, _ := createTestConn(t)
	defer conn.Close()
	app := appPkg.NewApp(client)

	returnValue := "foo"

	cb := rpc.NewRPC(
		"test_callback_1",
		func(args *rpc.Invocation) (interface{}, error) {
			return returnValue, nil
		},
	)

	if err := rpc.Register(ctx, client, cb); err != nil {
		t.Fatal(err)
	}

	var result string
	if err := app.InvokeFunction("test_callback_1()", &result); err != nil {
		t.Fatal(err)
	}

	if result != returnValue {
		t.Fatalf("expected %q, got %v", returnValue, result)
	}
}

func TestRegisterCallback_WithError(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, conn, _ := createTestConn(t)
	defer conn.Close()

	app := appPkg.NewApp(client)

	errorString := "something went wrong in the callback"

	cb := rpc.NewRPC(
		"test_callback_2",
		func(args *rpc.Invocation) (interface{}, error) {
			return nil, errors.New(errorString)
		},
	)

	if err := rpc.Register(ctx, client, cb); err != nil {
		t.Fatal(err)
	}

	var result string
	err := app.InvokeFunction("test_callback_2()", &result)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.Error() != fmt.Sprintf("FAILED: %s", errorString) {
		t.Fatalf("expected %q, got %q", errorString, err)
	}
}

func TestRegisterCallback_WithArguments(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, conn, _ := createTestConn(t)
	defer conn.Close()

	app := appPkg.NewApp(client)

	var callbackReturnValue string

	cb := rpc.NewRPC(
		"test_callback_3",
		func(args *rpc.Invocation) (interface{}, error) {
			val, err := args.GetString("foo")
			if err != nil {
				return nil, err
			}

			callbackReturnValue = val
			return callbackReturnValue, nil
		},
		rpc.Arg{Name: "foo"},
	)

	if err := rpc.Register(ctx, client, cb); err != nil {
		t.Fatal(err)
	}

	var result string
	err := app.InvokeFunction(fmt.Sprintf("test_callback_3(foo: %q)", "bar"), &result)
	if err != nil {
		t.Fatal(err)
	}

	if result != callbackReturnValue {
		t.Fatalf("expected %q, got %v", callbackReturnValue, result)
	}
}
