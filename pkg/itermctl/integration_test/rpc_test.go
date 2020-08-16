// +build test_with_iterm

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"mrz.io/itermctl/pkg/itermctl"
	"mrz.io/itermctl/pkg/itermctl/internal/test"
	"testing"
)

func TestRegisterCallback(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := itermctl.GetCredentialsAndConnect(test.AppName(t), true)
	defer conn.Close()
	client := itermctl.NewClient(conn)
	app := itermctl.NewApp(client)

	returnValue := "foo"

	rpc := itermctl.Rpc{
		Name: "test_callback_1",
		Args: nil,
		F: func(args *itermctl.RpcInvocation) (interface{}, error) {
			return returnValue, nil
		},
	}

	if err = itermctl.RegisterRpc(ctx, client, rpc); err != nil {
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

	conn, err := itermctl.GetCredentialsAndConnect(test.AppName(t), true)
	defer conn.Close()
	client := itermctl.NewClient(conn)

	app := itermctl.NewApp(client)

	errorString := "something went wrong in the callback"

	rpc := itermctl.Rpc{
		Name: "test_callback_2",
		Args: nil,
		F: func(args *itermctl.RpcInvocation) (interface{}, error) {
			return nil, errors.New(errorString)
		},
	}

	if err := itermctl.RegisterRpc(ctx, client, rpc); err != nil {
		t.Fatal(err)
	}

	var result string
	err = app.InvokeFunction("test_callback_2()", &result)

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

	conn, err := itermctl.GetCredentialsAndConnect(test.AppName(t), true)
	defer conn.Close()
	client := itermctl.NewClient(conn)

	app := itermctl.NewApp(client)

	var callbackReturnValue string

	var args struct {
		Foo string
	}

	rpc := itermctl.Rpc{
		Name: "test_callback_3",
		Args: args,
		F: func(invocation *itermctl.RpcInvocation) (interface{}, error) {
			err := invocation.Args(&args)
			if err != nil {
				return nil, err
			}

			callbackReturnValue = args.Foo
			return callbackReturnValue, nil
		},
	}

	if err := itermctl.RegisterRpc(ctx, client, rpc); err != nil {
		t.Fatal(err)
	}

	var result string
	err = app.InvokeFunction(fmt.Sprintf("test_callback_3(foo: %q)", "bar"), &result)
	if err != nil {
		t.Fatal(err)
	}

	if result != callbackReturnValue {
		t.Fatalf("expected %q, got %v", callbackReturnValue, result)
	}
}
