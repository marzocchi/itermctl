package test

import (
	"fmt"
	"mrz.io/itermctl"
	"mrz.io/itermctl/iterm2"
	"sync"
	"testing"
)

func AppName(t *testing.T) string {
	return fmt.Sprintf("itermctl_%s", t.Name())
}

const profileName = "itermctl test profile"

var windows = make(map[string][]string)
var windowsLock = &sync.Mutex{}

func AssertNoLeftoverWindows(app *itermctl.App, t *testing.T) {
	sessions, err := app.ListSessions()
	if err != nil {
		t.Fatal(err)
	}

	existingWindows := make(map[string]struct{})
	for _, w := range sessions.Windows {
		existingWindows[w.GetWindowId()] = struct{}{}
	}

	if thisTestWindows, ok := windows[t.Name()]; ok {
		var leftOvers []string
		for _, w := range thisTestWindows {
			if _, ok := existingWindows[w]; ok {
				leftOvers = append(leftOvers, w)
			}
		}

		if len(leftOvers) > 0 {
			t.Fatalf("%s left %d open windows over: %v", t.Name(), len(leftOvers), leftOvers)
		}
	}
}

func CreateWindow(app *itermctl.App, t *testing.T) (*iterm2.CreateTabResponse, func()) {
	response, err := app.CreateTab("", 0, profileName)
	if err != nil {
		t.Fatal(err)
	}

	windowId := response.GetWindowId()

	windowsLock.Lock()
	defer windowsLock.Unlock()

	windows[t.Name()] = append(windows[t.Name()], windowId)

	return response, func() {
		err = app.CloseTerminalWindow(true, windowId)

		if err != nil {
			t.Fatal(err)
		}

		windowsLock.Lock()
		defer windowsLock.Unlock()

		delete(windows, t.Name())
	}
}
