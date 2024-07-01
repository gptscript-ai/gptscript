package github

import (
	"context"
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

const (
	GithubPrefix      = "github.com/"
	githubRepoURL     = "https://github.com/%s/%s.git"
	githubDownloadURL = "https://raw.githubusercontent.com/%s/%s/%s/%s"
	githubCommitURL   = "https://api.github.com/repos/%s/%s/commits/%s"
)

var (
	githubAuthToken = os.Getenv("GITHUB_AUTH_TOKEN")
	log             = mvl.Package()
)

func init() {
	loader.AddVSC(Load)
}

func getCommitLsRemote(ctx context.Context, account, repo, ref string) (string, error) {
	url := fmt.Sprintf(githubRepoURL, account, repo)
	return git.LsRemote(ctx, url, ref)
}

// regexp to match a git commit id
var commitRegexp = regexp.MustCompile("^[a-f0-9]{40}$")

func getCommit(ctx context.Context, account, repo, ref string) (string, error) {
	if commitRegexp.MatchString(ref) {
		return ref, nil
	}

	url := fmt.Sprintf(githubCommitURL, account, repo, ref)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request of %s/%s at %s: %w", account, repo, url, err)
	}

	if githubAuthToken != "" {
		req.Header.Add("Authorization", "Bearer "+githubAuthToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	} else if resp.StatusCode != http.StatusOK {
		c, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		commit, fallBackErr := getCommitLsRemote(ctx, account, repo, ref)
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

func Load(ctx context.Context, _ *cache.Client, urlName string) (string, string, *types.Repo, bool, error) {
	if !strings.HasPrefix(urlName, GithubPrefix) {
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

	ref, err := getCommit(ctx, account, repo, ref)
	if err != nil {
		return "", "", nil, false, err
	}

	downloadURL := fmt.Sprintf(githubDownloadURL, account, repo, ref, path)
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
			testURL = fmt.Sprintf(githubDownloadURL, account, repo, ref, testPath)
			if i == len(types.DefaultFiles)-1 {
				// no reason to test the last one, we are just going to use it. Being that the default list is only
				// two elements this loop could have been one check, but hey over-engineered code ftw.
				break
			}
			if resp, err := http.Head(testURL); err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode == 200 {
					break
				}
			}
		}
		downloadURL = testURL
		path = testPath
	}

	return downloadURL, githubAuthToken, &types.Repo{
		VCS:      "git",
		Root:     fmt.Sprintf(githubRepoURL, account, repo),
		Path:     gpath.Dir(path),
		Name:     gpath.Base(path),
		Revision: ref,
	}, true, nil
}
