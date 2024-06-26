package remote

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/gptscript-ai/gptscript/pkg/cache"
	gcontext "github.com/gptscript-ai/gptscript/pkg/context"
	"github.com/gptscript-ai/gptscript/pkg/credentials"
	"github.com/gptscript-ai/gptscript/pkg/engine"
	env2 "github.com/gptscript-ai/gptscript/pkg/env"
	"github.com/gptscript-ai/gptscript/pkg/loader"
	"github.com/gptscript-ai/gptscript/pkg/openai"
	"github.com/gptscript-ai/gptscript/pkg/prompt"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

type Client struct {
	clientsLock sync.Mutex
	cache       *cache.Client
	clients     map[string]*openai.Client
	models      map[string]*openai.Client
	runner      *runner.Runner
	envs        []string
	credStore   credentials.CredentialStore
}

func New(r *runner.Runner, envs []string, cache *cache.Client, credStore credentials.CredentialStore) *Client {
	return &Client{
		cache:     cache,
		runner:    r,
		envs:      envs,
		credStore: credStore,
	}
}

func (c *Client) Call(ctx context.Context, messageRequest types.CompletionRequest, status chan<- types.CompletionStatus) (*types.CompletionMessage, error) {
	c.clientsLock.Lock()
	client, ok := c.models[messageRequest.Model]
	c.clientsLock.Unlock()

	if !ok {
		return nil, fmt.Errorf("failed to find remote model %s", messageRequest.Model)
	}

	_, modelName := types.SplitToolRef(messageRequest.Model)
	messageRequest.Model = modelName
	return client.Call(ctx, messageRequest, status)
}

func (c *Client) ListModels(ctx context.Context, providers ...string) (result []string, _ error) {
	for _, provider := range providers {
		client, err := c.load(ctx, provider)
		if err != nil {
			return nil, err
		}
		models, err := client.ListModels(ctx, "")
		if err != nil {
			return nil, err
		}
		for _, model := range models {
			result = append(result, model+" from "+provider)
		}
	}

	sort.Strings(result)
	return
}

func (c *Client) Supports(ctx context.Context, modelName string) (bool, error) {
	toolName, modelNameSuffix := types.SplitToolRef(modelName)
	if modelNameSuffix == "" {
		return false, nil
	}

	client, err := c.load(ctx, toolName)
	if err != nil {
		return false, err
	}

	c.clientsLock.Lock()
	defer c.clientsLock.Unlock()

	if c.models == nil {
		c.models = map[string]*openai.Client{}
	}

	c.models[modelName] = client
	return true, nil
}

func isHTTPURL(toolName string) bool {
	return strings.HasPrefix(toolName, "http://") ||
		strings.HasPrefix(toolName, "https://")
}

func (c *Client) clientFromURL(ctx context.Context, apiURL string) (*openai.Client, error) {
	parsed, err := url.Parse(apiURL)
	if err != nil {
		return nil, err
	}
	env := "GPTSCRIPT_PROVIDER_" + env2.ToEnvLike(parsed.Hostname()) + "_API_KEY"
	key := os.Getenv(env)

	if key == "" && !isLocalhost(apiURL) {
		var err error
		key, err = c.retrieveAPIKey(ctx, env, apiURL)
		if err != nil {
			return nil, err
		}
	}

	return openai.NewClient(ctx, c.credStore, openai.Options{
		BaseURL: apiURL,
		Cache:   c.cache,
		APIKey:  key,
	})
}

func (c *Client) load(ctx context.Context, toolName string) (*openai.Client, error) {
	c.clientsLock.Lock()
	defer c.clientsLock.Unlock()

	client, ok := c.clients[toolName]
	if ok {
		return client, nil
	}

	if c.clients == nil {
		c.clients = make(map[string]*openai.Client)
	}

	if isHTTPURL(toolName) {
		remoteClient, err := c.clientFromURL(ctx, toolName)
		if err != nil {
			return nil, err
		}
		c.clients[toolName] = remoteClient
		return remoteClient, nil
	}

	prg, err := loader.Program(ctx, toolName, "", loader.Options{
		Cache: c.cache,
	})
	if err != nil {
		return nil, err
	}

	url, err := c.runner.Run(engine.WithToolCategory(ctx, engine.ProviderToolCategory), prg.SetBlocking(), c.envs, "")
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(url, "/") {
		url += "v1"
	} else {
		url += "/v1"
	}

	client, err = openai.NewClient(ctx, c.credStore, openai.Options{
		BaseURL:  url,
		Cache:    c.cache,
		CacheKey: prg.EntryToolID,
	})
	if err != nil {
		return nil, err
	}

	c.clients[toolName] = client
	return client, nil
}

func (c *Client) retrieveAPIKey(ctx context.Context, env, url string) (string, error) {
	return prompt.GetModelProviderCredential(ctx, c.credStore, url, env, fmt.Sprintf("Please provide your API key for %s", url), append(gcontext.GetEnv(ctx), c.envs...))
}

func isLocalhost(url string) bool {
	return strings.HasPrefix(url, "http://localhost") || strings.HasPrefix(url, "http://127.0.0.1") ||
		strings.HasPrefix(url, "https://localhost") || strings.HasPrefix(url, "https://127.0.0.1")
}
