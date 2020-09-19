package main

import (
	"fmt"
	"mrz.io/itermctl/pkg/itermctl"
)

func main() {
	conn, err := itermctl.GetCredentialsAndConnect("itermctl_alert_example", true)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	app, err := itermctl.NewApp(conn)
	if err != nil {
		panic(err)
	}

	windowId, err := app.ActiveTerminalWindowId()
	if err != nil {
		panic(err)
	}

	userInput, err := app.GetText(itermctl.TextInputAlert{
		Title:        "Type something",
		Subtitle:     "Type something in the field below",
		Placeholder:  "Placeholder for your text",
		DefaultValue: "",
	}, windowId)

	if err != nil {
		panic(err)
	}

	button, err := app.ShowAlert(itermctl.Alert{
		Title:    "Test",
		Subtitle: fmt.Sprintf("You typed: %s", userInput),
	}, windowId)

	if err != nil {
		panic(err)
	}

	fmt.Printf("button: %s\n", button)
}
