package loader

import (
	"context"
	"fmt"
	"net/http"
	url2 "net/url"
	"path"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/types"
)

type VCSLookup func(string) (string, *types.Repo, bool, error)

var vcsLookups []VCSLookup

func AddVSC(lookup VCSLookup) {
	vcsLookups = append(vcsLookups, lookup)
}

func loadURL(ctx context.Context, base *source, name string) (*source, bool, error) {
	var (
		repo     *types.Repo
		url      = name
		relative = strings.HasPrefix(name, ".") || !strings.Contains(name, "/")
	)

	if base.Path != "" && relative {
		url = base.Path + "/" + name
	}

	if base.Repo != nil {
		newRepo := *base.Repo
		newPath := path.Join(newRepo.Path, name)
		newRepo.Path = path.Dir(newPath)
		newRepo.Name = path.Base(newPath)
		repo = &newRepo
	}

	if repo == nil || !relative {
		for _, vcs := range vcsLookups {
			newURL, newRepo, ok, err := vcs(name)
			if err != nil {
				return nil, false, err
			} else if ok {
				repo = newRepo
				url = newURL
				break
			}
		}
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, false, nil
	}

	parsed, err := url2.Parse(url)
	if err != nil {
		return nil, false, err
	}

	pathURL := *parsed
	pathURL.Path = path.Dir(parsed.Path)
	pathString := pathURL.String()
	name = path.Base(parsed.Path)
	url = pathString + "/" + name

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false, err
	} else if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("error loading %s: %s", url, resp.Status)
	}

	log.Debugf("opened %s", url)

	return &source{
		Content:  resp.Body,
		Remote:   true,
		Path:     pathString,
		Name:     name,
		Location: url,
		Repo:     repo,
	}, true, nil
}
