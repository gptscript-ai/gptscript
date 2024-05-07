package loader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	url2 "net/url"
	"path"
	"strings"
	"time"

	"github.com/gptscript-ai/gptscript/pkg/cache"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

type VCSLookup func(context.Context, *cache.Client, string) (string, *types.Repo, bool, error)

var vcsLookups []VCSLookup

func AddVSC(lookup VCSLookup) {
	vcsLookups = append(vcsLookups, lookup)
}

type cacheKey struct {
	Name string
	Path string
	Repo *types.Repo
}

type cacheValue struct {
	Source *source
	Time   time.Time
}

func loadURL(ctx context.Context, cache *cache.Client, base *source, name string) (*source, bool, error) {
	var (
		repo      *types.Repo
		url       = name
		relative  = strings.HasPrefix(name, ".") || !strings.Contains(name, "/")
		cachedKey = cacheKey{
			Name: name,
			Path: base.Path,
			Repo: base.Repo,
		}
		cachedValue cacheValue
	)

	if ok, err := cache.Get(ctx, cachedKey, &cachedValue); err != nil {
		return nil, false, err
	} else if ok && time.Since(cachedValue.Time) < CacheTimeout {
		return cachedValue.Source, true, nil
	}

	if base.Path != "" && relative {
		// Don't use path.Join because this is a URL and will break the :// protocol by cleaning it
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
			newURL, newRepo, ok, err := vcs(ctx, cache, name)
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

	// Append to pathString name. This is not the same as the original URL. This is an attempt to end up
	// with a clean URL with no ../ in it.
	if strings.HasSuffix(pathString, "/") {
		url = pathString + name
	} else {
		url = pathString + "/" + name
	}

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

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("error loading %s: %v", url, err)
	}

	result := &source{
		Content:  data,
		Remote:   true,
		Path:     pathString,
		Name:     name,
		Location: url,
		Repo:     repo,
	}

	if err := cache.Store(ctx, cachedKey, cacheValue{
		Source: result,
		Time:   time.Now(),
	}); err != nil {
		return nil, false, err
	}

	return result, true, nil
}
