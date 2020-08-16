package test

import (
	"fmt"
	"testing"
)

func AppName(t *testing.T) string {
	return fmt.Sprintf("itermctl_%s", t.Name())
}
