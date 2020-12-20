package env

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
)

var ErrNoSessionId = fmt.Errorf("the ITERM_SESSION_ID environment variable is not set")
var ErrNoCookie = fmt.Errorf("the ITERM2_COOKIE environment variable is not set")
var ErrNoKey = fmt.Errorf("the ITERM2_KEY environment variable is not set")

// Session contains session information as reported by the ITERM_SESSION_ID environment variable.
type Session struct {
	Id          string
	WindowIndex int
	TabIndex    int
}

// CurrentSession() parses the ITERM_SESSION_ID environment variable and returns a Session or ErrNoSessionId if the
// env var is not set.
func CurrentSession() (Session, error) {
	v := os.Getenv("ITERM_SESSION_ID")
	if v == "" {
		return Session{}, ErrNoSessionId
	}

	re := regexp.MustCompile("^w(\\d+)t(\\d+)p(\\d+):(.*)$")

	matches := re.FindStringSubmatch(v)

	var err error
	var w, t int

	if w, err = strconv.Atoi(matches[1]); err != nil {
		return Session{}, fmt.Errorf("get session: %w", err)
	}

	if t, err = strconv.Atoi(matches[2]); err != nil {
		return Session{}, fmt.Errorf("get session: %w", err)
	}

	return Session{Id: matches[4], WindowIndex: w, TabIndex: t}, nil
}

// CookieAndKey retrieves the cookie and key from the environment.
func CookieAndKey() (string, string, error) {
	cookie := os.Getenv("ITERM2_COOKIE")

	if cookie == "" {
		return "", "", ErrNoCookie
	}

	key := os.Getenv("ITERM2_KEY")

	if key == "" {
		return "", "", ErrNoKey
	}

	return cookie, key, nil
}
