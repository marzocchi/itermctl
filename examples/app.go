package main

import (
	"fmt"
	"mrz.io/itermctl/pkg/itermctl"
	"time"
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

	for {
		fmt.Printf("App active: %t\n", app.Active())
		s := app.ActiveSession()

		if s != nil {
			fmt.Printf("session ID: %s\n", s.Id())
	 	} else {
	 		fmt.Printf("session is nil\n")
		}

		<-time.After(1 * time.Second)
	}
}
