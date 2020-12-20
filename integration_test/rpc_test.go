// +build test_with_iterm

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"mrz.io/itermctl/rpc"
	"testing"
)

func TestRegisterCallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	returnValue := "foo"

	if err := rpc.Register(ctx, conn, rpc.RPC{
		Name: "test_callback_1",
		Args: nil,
		Function: func(args *rpc.Invocation) (interface{}, error) {
			return returnValue, nil
		},
	}); err != nil {
		t.Fatal(err)
	}

	var result string
	if err := conn.InvokeFunction("test_callback_1()", &result); err != nil {
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

	errorString := "something went wrong in the callback"

	if err := rpc.Register(ctx, conn, rpc.RPC{
		Name: "test_callback_2",
		Args: nil,
		Function: func(args *rpc.Invocation) (interface{}, error) {
			return nil, errors.New(errorString)
		},
	}); err != nil {
		t.Fatal(err)
	}

	var result string
	err := conn.InvokeFunction("test_callback_2()", &result)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.Error() != fmt.Sprintf("FAILED: %s", errorString) {
		t.Fatalf("expected %q, got %q", errorString, err)
	}
}

func TestRegisterCallback_WithArguments(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var callbackReturnValue string

	var args struct {
		Foo string
	}

	if err := rpc.Register(ctx, conn, rpc.RPC{
		Name: "test_callback_3",
		Args: args,
		Function: func(invocation *rpc.Invocation) (interface{}, error) {
			err := invocation.Args(&args)
			if err != nil {
				return nil, err
			}

			callbackReturnValue = args.Foo
			return callbackReturnValue, nil
		},
	}); err != nil {
		t.Fatal(err)
	}

	var result string
	if err := conn.InvokeFunction(fmt.Sprintf("test_callback_3(foo: %q)", "bar"), &result); err != nil {
		t.Fatal(err)
	}

	if result != callbackReturnValue {
		t.Fatalf("expected %q, got %v", callbackReturnValue, result)
	}
}
