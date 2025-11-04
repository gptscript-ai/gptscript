//nolint:revive
package types

import (
	"testing"

	"github.com/hexops/autogold/v2"
)

func TestToolNormalizer(t *testing.T) {
	autogold.Expect("bobTool").Equal(t, ToolNormalizer("bob-tool"))
	autogold.Expect("bobTool").Equal(t, ToolNormalizer("bob_tool"))
	autogold.Expect("bobTool").Equal(t, ToolNormalizer("BOB tOOL"))
	autogold.Expect("barList").Equal(t, ToolNormalizer("bar_list from ./foo.yaml"))
	autogold.Expect("barList").Equal(t, ToolNormalizer("bar_list from ./foo.gpt"))
	autogold.Expect("write").Equal(t, ToolNormalizer("sys.write"))
	autogold.Expect("gpt4VVision").Equal(t, ToolNormalizer("github.com/gptscript-ai/gpt4-v-vision"))
	autogold.Expect("foo").Equal(t, ToolNormalizer("./foo/tool.gpt"))
	autogold.Expect("tool").Equal(t, ToolNormalizer("./tool.gpt"))
	autogold.Expect("tool").Equal(t, ToolNormalizer(".a/tool.gpt"))
	autogold.Expect("ab").Equal(t, ToolNormalizer(".ab/tool.gpt"))
	autogold.Expect("tool").Equal(t, ToolNormalizer(".../tool.gpt"))
}

func TestParse(t *testing.T) {
	tool, subTool := SplitToolRef("a from b with x")
	autogold.Expect([]string{"b", "a"}).Equal(t, []string{tool, subTool})

	tool, subTool = SplitToolRef("a from b with x as other")
	autogold.Expect([]string{"b", "a"}).Equal(t, []string{tool, subTool})

	tool, subTool = SplitToolRef("a with x")
	autogold.Expect([]string{"a", ""}).Equal(t, []string{tool, subTool})

	tool, subTool = SplitToolRef("a with x as other")
	autogold.Expect([]string{"a", ""}).Equal(t, []string{tool, subTool})
}
