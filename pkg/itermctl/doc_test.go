package itermctl

import (
	"context"
	"fmt"
)

func ExampleRegisterRpc() {
	var client *Client

	type Args struct {
		BoolArg   bool    `arg.name:"bool_arg"`
		NumberArg float64 `arg.name:"number_arg"`
		// this will be injected with the value of the current session ID when invoked by iTerm2
		StringArg string `arg.name:"string_arg" arg.ref:"id"`
	}

	rpc := Rpc{
		Name: "itermctl_rpc_example",
		Args: Args{
			BoolArg:   true,
			StringArg: "some string",
			NumberArg: 42.0,
		},
		F: func(invocation *RpcInvocation) (interface{}, error) {
			args := Args{}
			if err := invocation.Args(args); err != nil {
				return nil, err
			}

			// do stuff with args.StringArg, etc. and maybe return something
			return nil, nil
		},
	}

	_ = RegisterRpc(context.Background(), client, rpc)
}

func ExampleRegisterStatusBarComponent() {
	var client *Client

	type Knobs struct {
		Checkbox bool    `knob.name:"Checkbox knob" json:"checkbox"`
		Text     string  `knob.name:"Text knob" knob.placeholder:"Enter some text" json:"text"`
		Number   float64 `knob.name:"Number knob" json:"number"`
	}

	cmp := StatusBarComponent{
		ShortDescription: "Component example",
		Description:      "Component example",
		Exemplar:         "[component]",
		UpdateCadence:    0,
		Identifier:       "io.mrz.itermctl.example.component",
		Rpc: Rpc{
			Name: "itermctl_component_example",
			F: func(invocation *RpcInvocation) (interface{}, error) {
				knobs := Knobs{}

				if err := invocation.Knobs(&knobs); err != nil {
					return nil, err
				}

				return fmt.Sprintf("checkbox: %t, number: %f, text:%s", knobs.Checkbox, knobs.Number, knobs.Text), nil
			},
		},
		Knobs: Knobs{
			// Values given here are used as default values in the configuration panel
			Checkbox: true,
			Text:     "some text",
			Number:   42.0,
		},
	}

	_ = RegisterStatusBarComponent(context.Background(), client, cmp)
}

func ExampleRegisterSessionTitleProvider() {
	var client *Client

	var args struct {
		SessionId string `arg.name:"session_id" arg.ref:"id"`
	}

	tp := TitleProvider{
		DisplayName: "Title Provider",
		Identifier:  "io.mrz.itermctl.example.titleprovider",
		Rpc: Rpc{
			Name: "itermctl_title_provider_example",
			Args: args,
			F: func(invocation *RpcInvocation) (interface{}, error) {
				err := invocation.Args(&args)
				if err != nil {
					return nil, err
				}

				return fmt.Sprintf("Title for session %q", args.SessionId), nil
			},
		},
	}

	_ = RegisterSessionTitleProvider(context.Background(), client, tp)
}
