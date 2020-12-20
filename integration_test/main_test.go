package integration_test

import (
	"fmt"
	"mrz.io/itermctl"
	"os"
	"testing"
)

var conn *itermctl.Connection
var app *itermctl.App

const profileName = "itermctl test profile"

func TestMain(m *testing.M) {
	var err error

	conn, err = itermctl.GetCredentialsAndConnect("itermctl_integration_test", true)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	app, err = itermctl.NewApp(conn)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	os.Exit(func() int {
		defer conn.Close()
		return m.Run()
	}())
}
