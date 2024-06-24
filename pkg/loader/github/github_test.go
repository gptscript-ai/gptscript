package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestLoad_GithubEnterprise(t *testing.T) {
	gheToken := "mytoken"
	os.Setenv("GH_ENTERPRISE_SKIP_VERIFY", "true")
	os.Setenv("GH_ENTERPRISE_TOKEN", gheToken)
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v3/repos/gptscript-ai/gptscript/commits/172dfb0":
			_, _ = w.Write([]byte(`{"sha": "172dfb00b48c6adbbaa7e99270933f95887d1b91"}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer s.Close()

	serverAddr := s.Listener.Addr().String()

	url, token, repo, ok, err := LoadWithConfig(context.Background(), nil, fmt.Sprintf("%s/gptscript-ai/gptscript/pkg/loader/testdata/tool@172dfb0", serverAddr), NewGithubEnterpriseConfig(serverAddr))
	require.NoError(t, err)
	assert.True(t, ok)
	autogold.Expect(fmt.Sprintf("https://raw.%s/gptscript-ai/gptscript/172dfb00b48c6adbbaa7e99270933f95887d1b91/pkg/loader/testdata/tool/tool.gpt", serverAddr)).Equal(t, url)
	autogold.Expect(&types.Repo{
		VCS: "git", Root: fmt.Sprintf("https://%s/gptscript-ai/gptscript.git", serverAddr),
		Path:     "pkg/loader/testdata/tool",
		Name:     "tool.gpt",
		Revision: "172dfb00b48c6adbbaa7e99270933f95887d1b91",
	}).Equal(t, repo)
	autogold.Expect(gheToken).Equal(t, token)

	url, token, repo, ok, err = Load(context.Background(), nil, "github.com/gptscript-ai/gptscript/pkg/loader/testdata/agent@172dfb0")
	require.NoError(t, err)
	assert.True(t, ok)
	autogold.Expect("https://raw.githubusercontent.com/gptscript-ai/gptscript/172dfb00b48c6adbbaa7e99270933f95887d1b91/pkg/loader/testdata/agent/agent.gpt").Equal(t, url)
	autogold.Expect(&types.Repo{
		VCS: "git", Root: "https://github.com/gptscript-ai/gptscript.git",
		Path:     "pkg/loader/testdata/agent",
		Name:     "agent.gpt",
		Revision: "172dfb00b48c6adbbaa7e99270933f95887d1b91",
	}).Equal(t, repo)
	autogold.Expect("").Equal(t, token)

	url, token, repo, ok, err = Load(context.Background(), nil, "github.com/gptscript-ai/gptscript/pkg/loader/testdata/bothtoolagent@172dfb0")
	require.NoError(t, err)
	assert.True(t, ok)
	autogold.Expect("https://raw.githubusercontent.com/gptscript-ai/gptscript/172dfb00b48c6adbbaa7e99270933f95887d1b91/pkg/loader/testdata/bothtoolagent/agent.gpt").Equal(t, url)
	autogold.Expect(&types.Repo{
		VCS: "git", Root: "https://github.com/gptscript-ai/gptscript.git",
		Path:     "pkg/loader/testdata/bothtoolagent",
		Name:     "agent.gpt",
		Revision: "172dfb00b48c6adbbaa7e99270933f95887d1b91",
	}).Equal(t, repo)
	autogold.Expect("").Equal(t, token)
}
