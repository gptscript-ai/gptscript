package loader

import (
	"context"
	"fmt"
	"net/http"
	url2 "net/url"
	"path/filepath"
	"strings"
)

type VCSLookup func(string) (string, *Repo, bool, error)

var vcsLookups []VCSLookup

func AddVSC(lookup VCSLookup) {
	vcsLookups = append(vcsLookups, lookup)
}

func loadURL(ctx context.Context, base *source, name string) (*source, bool, error) {
	var (
		repo *Repo
		url  = name
	)

	if base.Path != "" {
		url = base.Path + "/" + name
	}

	if base.Repo != nil {
		newRepo := *base.Repo
		newPath := filepath.Join(newRepo.Path, name)
		newRepo.Path = filepath.Dir(newPath)
		newRepo.Name = filepath.Base(newPath)
		repo = &newRepo
	}

	if repo == nil {
		for _, vcs := range vcsLookups {
			newURL, newRepo, ok, err := vcs(url)
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
	pathURL.Path = filepath.Dir(parsed.Path)
	path := pathURL.String()
	name = filepath.Base(parsed.Path)
	url = path + "/" + name

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
		Path:     path,
		Name:     name,
		Location: url,
		Repo:     repo,
	}, true, nil
}
