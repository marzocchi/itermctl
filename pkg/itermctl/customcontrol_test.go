package itermctl

import (
	"testing"
)

func TestCustomControlSequenceEscaper_Escape(t *testing.T) {
	expected := "\u001B]1337;Custom=id=foo:hello world\a"

	escaper := NewCustomControlSequenceEscaper("foo")
	str := escaper.Escape("hello %s", "world")

	if str != expected {
		t.Fatalf("expected %#v, got %#v", expected, str)
	}
}
