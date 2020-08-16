package rpc

import (
	"encoding/json"
	"fmt"
	iterm2 "mrz.io/itermctl/pkg/itermctl/proto"
	"testing"
)

func TestInvocation(t *testing.T) {
	expectedName := "test"
	expectedValueOfA := `a`
	expectedValueOfB := 42.0
	expectedValueOfC := map[string]map[string]int{
		"d": {
			"e": 42,
		},
	}

	invocation := NewInvocation(expectedName, map[string]string{
		"a": marshal(expectedValueOfA),
		"b": marshal(expectedValueOfB),
		"c": marshal(expectedValueOfC),
	})

	if invocation.Name() != expectedName {
		t.Fatalf("expected Name() = %q, got %q", invocation.Name(), expectedName)
	}

	valueOfA, err := invocation.GetString("a")
	if err != nil || valueOfA != expectedValueOfA {
		t.Fatalf("expected %q, got value = %q, err = %s", expectedValueOfA, valueOfA, err)
	}

	valueOfB, err := invocation.GetFloat64("b")
	if err != nil || valueOfB != expectedValueOfB {
		t.Fatalf("expected %f, got value = %f, err = %s", expectedValueOfB, valueOfB, err)
	}

	var valueOfC map[string]map[string]int
	err = invocation.Get("c", &valueOfC)
	if err != nil || valueOfC["d"]["e"] != expectedValueOfC["d"]["e"] {
		t.Fatalf("expected %v, got value %v, err = %s", expectedValueOfC, valueOfC, err)
	}
}

func Test_apply_success(t *testing.T) {
	requestId := "rpc-1234"
	invocationName := "test"

	argName := "a"
	argValue := "a"

	jsonValue := "\"a\""
	errorString := "something went wrong"
	expectedJsonError := "{\"reason\":\"something went wrong\"}"

	req := &iterm2.ServerOriginatedRPCNotification{
		RequestId: &requestId,
		Rpc: &iterm2.ServerOriginatedRPC{
			Name: &invocationName,
			Arguments: []*iterm2.ServerOriginatedRPC_RPCArgument{
				{
					Name:      &argName,
					JsonValue: &argValue,
				},
			},
		},
	}

	successFunc := func(i *Invocation) (interface{}, error) {
		return argValue, nil
	}

	failFunc := func(i *Invocation) (interface{}, error) {
		return nil, fmt.Errorf(errorString)
	}

	successResp := apply(NewRPC(invocationName, successFunc, Arg{Name: argName}), req)
	failResp := apply(NewRPC(invocationName, failFunc, Arg{Name: argName}), req)

	if successResp.GetJsonValue() != jsonValue {
		t.Fatalf("expected %#v, got %#v", jsonValue, successResp.GetJsonValue())
	}

	if failResp.GetJsonException() != expectedJsonError {
		t.Fatalf("expected %#v, got %#v", expectedJsonError, failResp.GetJsonException())
	}
}

func marshal(v interface{}) string {
	s, _ := json.Marshal(v)
	return string(s)
}
