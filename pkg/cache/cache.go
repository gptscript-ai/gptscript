package cache

import (
	"context"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/getkin/kin-openapi/openapi3"
	openai "github.com/gptscript-ai/chat-completion-client"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/gptscript-ai/gptscript/pkg/version"
)

type Client struct {
	dir  string
	noop bool
}

type Options struct {
	DisableCache bool   `usage:"Disable caching of LLM API responses"`
	CacheDir     string `usage:"Directory to store cache (default: $XDG_CACHE_HOME/gptscript)"`
}

func init() {
	gob.Register(openai.ChatCompletionRequest{})
	gob.Register(openapi3.Schema{})
}

func Complete(opts ...Options) (result Options) {
	for _, opt := range opts {
		result.CacheDir = types.FirstSet(opt.CacheDir, result.CacheDir)
		result.DisableCache = types.FirstSet(opt.DisableCache, result.DisableCache)
	}
	if result.CacheDir == "" {
		result.CacheDir = filepath.Join(xdg.CacheHome, version.ProgramName)
	} else if !filepath.IsAbs(result.CacheDir) {
		var err error
		result.CacheDir, err = makeAbsolute(result.CacheDir)
		if err != nil {
			result.CacheDir = filepath.Join(xdg.CacheHome, version.ProgramName)
		}
	}
	return
}

func makeAbsolute(path string) (string, error) {
	if strings.HasPrefix(path, "~"+string(filepath.Separator)) {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}

		return filepath.Join(usr.HomeDir, path[2:]), nil
	}
	return filepath.Abs(path)
}

type noCacheKey struct{}

func IsNoCache(ctx context.Context) bool {
	v, _ := ctx.Value(noCacheKey{}).(bool)
	return v
}

func WithNoCache(ctx context.Context) context.Context {
	return context.WithValue(ctx, noCacheKey{}, true)
}

func New(opts ...Options) (*Client, error) {
	opt := Complete(opts...)
	if err := os.MkdirAll(opt.CacheDir, 0755); err != nil {
		return nil, err
	}
	return &Client{
		dir:  opt.CacheDir,
		noop: opt.DisableCache,
	}, nil
}

func (c *Client) CacheDir() string {
	return c.dir
}

func (c *Client) cacheKey(key any) (string, error) {
	hash := sha256.New()
	hash.Write([]byte("v2"))
	if err := json.NewEncoder(hash).Encode(key); err != nil {
		return "", err
	}
	digest := hash.Sum(nil)
	return hex.EncodeToString(digest), nil
}

func (c *Client) Store(ctx context.Context, key, value any) error {
	if c == nil {
		return nil
	}

	if c.noop || IsNoCache(ctx) {
		keyValue, err := c.cacheKey(key)
		if err == nil {
			p := filepath.Join(c.dir, keyValue)
			if _, err := os.Stat(p); err == nil {
				_ = os.Remove(p)
			}
		}
		return nil
	}

	keyValue, err := c.cacheKey(key)
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(c.dir, keyValue))
	if err != nil {
		return err
	}
	defer f.Close()

	return gob.NewEncoder(f).Encode(value)
}

func (c *Client) Get(ctx context.Context, key, out any) (bool, error) {
	if c == nil || c.noop || IsNoCache(ctx) {
		return false, nil
	}

	keyValue, err := c.cacheKey(key)
	if err != nil {
		return false, err
	}

	f, err := os.Open(filepath.Join(c.dir, keyValue))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	defer f.Close()

	return gob.NewDecoder(f).Decode(out) == nil, nil
}
