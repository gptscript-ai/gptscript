package github

import (
	"context"
	"testing"

	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/hexops/autogold/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	url, _, repo, ok, err := Load(context.Background(), nil, "github.com/gptscript-ai/gptscript/pkg/loader/testdata/tool@172dfb0")
	require.NoError(t, err)
	assert.True(t, ok)
	autogold.Expect("https://raw.githubusercontent.com/gptscript-ai/gptscript/172dfb00b48c6adbbaa7e99270933f95887d1b91/pkg/loader/testdata/tool/tool.gpt").Equal(t, url)
	autogold.Expect(&types.Repo{
		VCS: "git", Root: "https://github.com/gptscript-ai/gptscript.git",
		Path:     "pkg/loader/testdata/tool",
		Name:     "tool.gpt",
		Revision: "172dfb00b48c6adbbaa7e99270933f95887d1b91",
	}).Equal(t, repo)

	url, _, repo, ok, err = Load(context.Background(), nil, "github.com/gptscript-ai/gptscript/pkg/loader/testdata/agent@172dfb0")
	require.NoError(t, err)
	assert.True(t, ok)
	autogold.Expect("https://raw.githubusercontent.com/gptscript-ai/gptscript/172dfb00b48c6adbbaa7e99270933f95887d1b91/pkg/loader/testdata/agent/agent.gpt").Equal(t, url)
	autogold.Expect(&types.Repo{
		VCS: "git", Root: "https://github.com/gptscript-ai/gptscript.git",
		Path:     "pkg/loader/testdata/agent",
		Name:     "agent.gpt",
		Revision: "172dfb00b48c6adbbaa7e99270933f95887d1b91",
	}).Equal(t, repo)

	url, _, repo, ok, err = Load(context.Background(), nil, "github.com/gptscript-ai/gptscript/pkg/loader/testdata/bothtoolagent@172dfb0")
	require.NoError(t, err)
	assert.True(t, ok)
	autogold.Expect("https://raw.githubusercontent.com/gptscript-ai/gptscript/172dfb00b48c6adbbaa7e99270933f95887d1b91/pkg/loader/testdata/bothtoolagent/agent.gpt").Equal(t, url)
	autogold.Expect(&types.Repo{
		VCS: "git", Root: "https://github.com/gptscript-ai/gptscript.git",
		Path:     "pkg/loader/testdata/bothtoolagent",
		Name:     "agent.gpt",
		Revision: "172dfb00b48c6adbbaa7e99270933f95887d1b91",
	}).Equal(t, repo)
}
