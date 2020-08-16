package main

import (
	"context"
	"encoding/json"
	"fmt"
	"mrz.io/itermctl/pkg/itermctl"
	"mrz.io/itermctl/pkg/itermctl/rpc"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cookie, err := itermctl.GetCookieAndKey("itermctl_statusbar_example", true)
	if err != nil {
		panic(err)
	}

	conn, err := itermctl.Connect(cookie)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := itermctl.NewClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signals
		cancel()
	}()

	cmp := rpc.StatusBarComponent{
		ShortDescription: "Example Clock",
		Description:      "Example Clock",
		Exemplar:         "2020-08-16 19:30:08 +0200 CEST",
		UpdateCadence:    1,
		Identifier:       "io.mrz.itermctl.example.clock",
		RPC: rpc.NewRPC(
			"itermctl_example_clock",
			func(args *rpc.Invocation) (interface{}, error) {
				knobs, err := args.GetString("knobs")
				if err != nil {
					return nil, err
				}

				var knobValues map[string]interface{}
				if err := json.Unmarshal([]byte(knobs), &knobValues); err != nil {
					return nil, err
				}

				for knob, value := range knobValues {
					fmt.Printf("%q: %v\n", knob, value)
				}

				fmt.Println("")

				return fmt.Sprintf("%s", time.Now().Round(1*time.Second)), nil
			},
		),
		Knobs: []rpc.Knob{
			rpc.NewCheckboxKnob("CheckboxKnob Test", "checkbox_test", true),
			rpc.NewStringKnob("StringKnob Test", "string_test", "default string"),
			rpc.NewPositiveFloatingPointKnob("FloatKnob Test", "float_test", 42.0),
		},
	}

	err = rpc.RegisterStatusBarComponent(ctx, client, cmp)
	if err != nil {
		panic(err)
	}

	<-ctx.Done()
}
