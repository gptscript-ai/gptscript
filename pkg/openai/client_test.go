package openai

import (
	"strings"
	"testing"
)

func Test_toolNormalizer(t *testing.T) {
	output := toolNormalizer("x/" + strings.Repeat("a", 64))
	if len(output) != 64 {
		t.Fatalf("not 64 characters %s %d", output, len(output))
	}
	exp := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-7f46b2b2"
	if output != exp {
		t.Fatalf("%s != %s", output, exp)
	}
}
