package rpc

import (
	"testing"
)

type argsStruct struct {
	SomeBool   bool    `arg.name:"some_bool"`
	SomeString string  `arg.name:"some_string" arg.reference:"id"`
	SomeFloat  float64 `arg.name:"some_float"`
}

func TestRpcInvocation_Args(t *testing.T) {
	inputArgs := map[string]string{
		"some_bool":   "true",
		"some_string": "\"string\"",
		"some_float":  "42.1",
	}

	args := Invocation{args: inputArgs}

	expected := &argsStruct{
		SomeBool:   true,
		SomeString: "string",
		SomeFloat:  42.1,
	}

	result := &argsStruct{}
	if err := args.Args(result); err != nil {
		t.Fatal(err)
	}

	if *result != *expected {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

type knobsStruct struct {
	SomeBool   bool    `knob.label:"Some Bool" json:"some_bool"`
	SomeString string  `knob.label:"Some String" json:"some_string"`
	SomeFloat  float64 `knob.label:"Some Float" json:"some_float"`
}

func TestRpcInvocation_Knobs(t *testing.T) {
	inputArgs := map[string]string{
		"knobs": "\"{\\\"some_bool\\\": true, \\\"some_string\\\": \\\"string\\\", \\\"some_float\\\": 42.1}\"",
	}

	args := Invocation{args: inputArgs}

	expected := &knobsStruct{
		SomeBool:   true,
		SomeString: "string",
		SomeFloat:  42.1,
	}

	result := &knobsStruct{}
	if err := args.Knobs(result); err != nil {
		t.Fatal(err)
	}

	if *result != *expected {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}
