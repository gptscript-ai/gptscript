package github

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	gpath "path"
	"regexp"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/cache"
	"github.com/gptscript-ai/gptscript/pkg/loader"
	"github.com/gptscript-ai/gptscript/pkg/mvl"
	"github.com/gptscript-ai/gptscript/pkg/repos/git"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

type Config struct {
	Prefix      string
	RepoURL     string
	DownloadURL string
	CommitURL   string
	AuthToken   string
}

var (
	log                 = mvl.Package()
	defaultGithubConfig = &Config{
		Prefix:      "github.com/",
		RepoURL:     "https://github.com/%s/%s.git",
		DownloadURL: "https://raw.githubusercontent.com/%s/%s/%s/%s",
		CommitURL:   "https://api.github.com/repos/%s/%s/commits/%s",
		AuthToken:   os.Getenv("GITHUB_AUTH_TOKEN"),
	}
)

func init() {
	loader.AddVSC(Load)
}

func getCommitLsRemote(ctx context.Context, account, repo, ref string, config *Config) (string, error) {
	url := fmt.Sprintf(config.RepoURL, account, repo)
	return git.LsRemote(ctx, url, ref)
}

// regexp to match a git commit id
var commitRegexp = regexp.MustCompile("^[a-f0-9]{40}$")

func getCommit(ctx context.Context, account, repo, ref string, config *Config) (string, error) {
	if commitRegexp.MatchString(ref) {
		return ref, nil
	}

	url := fmt.Sprintf(config.CommitURL, account, repo, ref)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request of %s/%s at %s: %w", account, repo, url, err)
	}

	if config.AuthToken != "" {
		req.Header.Add("Authorization", "Bearer "+config.AuthToken)
	}

	client := http.DefaultClient
	if req.Host == config.Prefix && strings.ToLower(os.Getenv("GH_ENTERPRISE_SKIP_VERIFY")) == "true" {
		client = &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	} else if resp.StatusCode != http.StatusOK {
		c, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		commit, fallBackErr := getCommitLsRemote(ctx, account, repo, ref, config)
		if fallBackErr == nil {
			return commit, nil
		}
		return "", fmt.Errorf("failed to get GitHub commit of %s/%s at %s (fallback error %v): %s %s",
			account, repo, ref, fallBackErr, resp.Status, c)
	}
	defer resp.Body.Close()

	var commit struct {
		SHA string `json:"sha,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&commit); err != nil {
		return "", fmt.Errorf("failed to decode GitHub commit of %s/%s at %s: %w", account, repo, url, err)
	}

	log.Debugf("loaded github commit of %s/%s at %s as %q", account, repo, url, commit.SHA)

	if commit.SHA == "" {
		return "", fmt.Errorf("failed to find commit in response of %s, got empty string", url)
	}

	return commit.SHA, nil
}

func LoaderForPrefix(prefix string) func(context.Context, *cache.Client, string) (string, string, *types.Repo, bool, error) {
	return func(ctx context.Context, c *cache.Client, urlName string) (string, string, *types.Repo, bool, error) {
		return LoadWithConfig(ctx, c, urlName, NewGithubEnterpriseConfig(prefix))
	}
}

func Load(ctx context.Context, c *cache.Client, urlName string) (string, string, *types.Repo, bool, error) {
	return LoadWithConfig(ctx, c, urlName, defaultGithubConfig)
}

func NewGithubEnterpriseConfig(prefix string) *Config {
	return &Config{
		Prefix:      prefix,
		RepoURL:     fmt.Sprintf("https://%s/%%s/%%s.git", prefix),
		DownloadURL: fmt.Sprintf("https://raw.%s/%%s/%%s/%%s/%%s", prefix),
		CommitURL:   fmt.Sprintf("https://%s/api/v3/repos/%%s/%%s/commits/%%s", prefix),
		AuthToken:   os.Getenv("GH_ENTERPRISE_TOKEN"),
	}
}

func LoadWithConfig(ctx context.Context, _ *cache.Client, urlName string, config *Config) (string, string, *types.Repo, bool, error) {
	if !strings.HasPrefix(urlName, config.Prefix) {
		return "", "", nil, false, nil
	}

	url, ref, _ := strings.Cut(urlName, "@")
	if ref == "" {
		ref = "HEAD"
	}

	parts := strings.Split(url, "/")
	// Must be at least 3 parts github.com/ACCOUNT/REPO[/FILE]
	if len(parts) < 3 {
		return "", "", nil, false, nil
	}

	account, repo := parts[1], parts[2]
	path := strings.Join(parts[3:], "/")

	ref, err := getCommit(ctx, account, repo, ref, config)
	if err != nil {
		return "", "", nil, false, err
	}

	downloadURL := fmt.Sprintf(config.DownloadURL, account, repo, ref, path)
	if path == "" || path == "/" || !strings.Contains(parts[len(parts)-1], ".") {
		var (
			testPath string
			testURL  string
		)
		for i, ext := range types.DefaultFiles {
			if strings.HasSuffix(path, "/") {
				testPath = path + ext
			} else {
				testPath = path + "/" + ext
			}
			testURL = fmt.Sprintf(config.DownloadURL, account, repo, ref, testPath)
			if i == len(types.DefaultFiles)-1 {
				// no reason to test the last one, we are just going to use it. Being that the default list is only
				// two elements this loop could have been one check, but hey over-engineered code ftw.
				break
			}
			headReq, err := http.NewRequest("HEAD", testURL, nil)
			if err != nil {
				break
			}
			if config.AuthToken != "" {
				headReq.Header.Add("Authorization", "Bearer "+config.AuthToken)
			}
			if resp, err := http.DefaultClient.Do(headReq); err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode == 200 {
					break
				}
			}
		}
		downloadURL = testURL
		path = testPath
	}

	return downloadURL, config.AuthToken, &types.Repo{
		VCS:      "git",
		Root:     fmt.Sprintf(config.RepoURL, account, repo),
		Path:     gpath.Dir(path),
		Name:     gpath.Base(path),
		Revision: ref,
	}, true, nil
}
