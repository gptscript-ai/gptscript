package types

import (
	"testing"

	"github.com/hexops/autogold/v2"
)

func TestToolNormalizer(t *testing.T) {
	autogold.Expect("bobTool").Equal(t, ToolNormalizer("bob-tool"))
	autogold.Expect("bobTool").Equal(t, ToolNormalizer("bob_tool"))
	autogold.Expect("bobTool").Equal(t, ToolNormalizer("BOB tOOL"))
}
