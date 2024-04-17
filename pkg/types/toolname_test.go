package types

import (
	"testing"

	"github.com/hexops/autogold/v2"
)

func TestToolNormalizer(t *testing.T) {
	autogold.Expect("bob_tool").Equal(t, ToolNormalizer("bob-tool"))
}
