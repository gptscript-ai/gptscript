package loader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	url2 "net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/gptscript-ai/gptscript/pkg/cache"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

type VCSLookup func(context.Context, *cache.Client, string) (string, string, *types.Repo, bool, error)

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

func (c *cacheKey) isStatic() bool {
	return c.Repo != nil &&
		c.Repo.Revision != "" &&
		stableRef.MatchString(c.Repo.Revision)
}

var stableRef = regexp.MustCompile("^([a-f0-9]{7,40}$|v[0-9]|[0-9])")

func loadURL(ctx context.Context, cache *cache.Client, base *source, name string) (*source, bool, error) {
	var (
		repo        *types.Repo
		url         = name
		bearerToken = ""
		relative    = strings.HasPrefix(name, ".") || !strings.Contains(name, "/")
		cachedKey   = cacheKey{
			Name: name,
			Path: base.Path,
			Repo: base.Repo,
		}
		cachedValue cacheValue
	)

	if cachedKey.Repo == nil {
		if _, rev, ok := strings.Cut(name, "@"); ok && stableRef.MatchString(rev) {
			cachedKey.Repo = &types.Repo{
				Revision: rev,
			}
		}
	}
	if cachedKey.Path == "" {
		cachedKey.Path = "."
	}

	if ok, err := cache.Get(ctx, cachedKey, &cachedValue); err != nil {
		return nil, false, err
	} else if ok && (cachedKey.isStatic() || time.Since(cachedValue.Time) < CacheTimeout) {
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
			newURL, newBearer, newRepo, ok, err := vcs(ctx, cache, name)
			if err != nil {
				return nil, false, err
			} else if ok {
				repo = newRepo
				url = newURL
				bearerToken = newBearer
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

	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	data, defaulted, err := getWithDefaults(req)
	if err != nil {
		return nil, false, fmt.Errorf("error loading %s: %v", url, err)
	}

	if defaulted != "" {
		pathString = url
		name = defaulted
		if repo != nil {
			repo.Path = path.Join(repo.Path, repo.Name)
			repo.Name = defaulted
		}
	}

	log.Debugf("opened %s", url)

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

func getWithDefaults(req *http.Request) ([]byte, string, error) {
	originalPath := req.URL.Path

	// First, try to get the original path as is. It might be an OpenAPI definition.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		toolBytes, err := io.ReadAll(resp.Body)
		return toolBytes, "", err
	}

	base := path.Base(originalPath)
	if strings.Contains(base, ".") {
		return nil, "", fmt.Errorf("error loading %s: %s", req.URL.String(), resp.Status)
	}

	for i, def := range types.DefaultFiles {
		req.URL.Path = path.Join(originalPath, def)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound && i != len(types.DefaultFiles)-1 {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, "", fmt.Errorf("error loading %s: %s", req.URL.String(), resp.Status)
		}

		data, err := io.ReadAll(resp.Body)
		return data, def, err
	}

	panic("unreachable")
}

func ContentFromURL(url string, disableCache bool) (string, error) {
	cache, err := cache.New(cache.Options{
		DisableCache: disableCache,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create cache: %w", err)
	}

	source, ok, err := loadURL(context.Background(), cache, &source{}, url)
	if err != nil {
		return "", fmt.Errorf("failed to load %s: %w", url, err)
	}

	if !ok {
		return "", nil
	}

	return string(source.Content), nil
}
