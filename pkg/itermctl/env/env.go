package env

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
)

type credentials struct {
	cookie string
	key    string
}

func (c credentials) Key() string {
	return c.key
}

func (c credentials) Cookie() string {
	return c.cookie
}

// CurrentSession contains session information as reported by the ITERM_SESSION_ID environment variable.
type CurrentSession struct {
	SessionId   string
	WindowIndex int
	TabIndex    int
}

// Session() parses the ITERM_SESSION_ID environment variable into a CurrentSession.
func Session() (CurrentSession, error) {
	v := os.Getenv("ITERM_SESSION_ID")
	if v == "" {
		return CurrentSession{}, fmt.Errorf("the ITERM_SESSION_ID environment variable is not set")
	}

	re := regexp.MustCompile("^w(\\d+)t(\\d+)p(\\d+):(.*)$")

	matches := re.FindStringSubmatch(v)

	var err error
	var w, t int

	if w, err = strconv.Atoi(matches[1]); err != nil {
		return CurrentSession{}, fmt.Errorf("get session: %w", err)
	}

	if t, err = strconv.Atoi(matches[2]); err != nil {
		return CurrentSession{}, fmt.Errorf("get session: %w", err)
	}

	return CurrentSession{SessionId: matches[4], WindowIndex: w, TabIndex: t}, nil
}

// CookieAndKey retrieves the cookie and key from the environment. No error but nil is returned if the ITERM2_COOKIE
// and ITERM2_KEY environment variables are not set.
func CookieAndKey() (string, string, error) {
	cookie := os.Getenv("ITERM2_COOKIE")

	if cookie == "" {
		return "", "", fmt.Errorf("ITERM2_COOKIE is not set")
	}

	key := os.Getenv("ITERM2_KEY")

	if key == "" {
		return "", "", fmt.Errorf("ITERM2_KEY is not set")
	}

	return cookie, key, nil
}
