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
	modelsLock      sync.Mutex
	cache           *cache.Client
	modelToProvider map[string]string
	runner          *runner.Runner
	envs            []string
	credStore       credentials.CredentialStore
	defaultProvider string
}

func New(r *runner.Runner, envs []string, cache *cache.Client, credStore credentials.CredentialStore, defaultProvider string) *Client {
	return &Client{
		cache:           cache,
		runner:          r,
		envs:            envs,
		credStore:       credStore,
		defaultProvider: defaultProvider,
	}
}

func (c *Client) Call(ctx context.Context, messageRequest types.CompletionRequest, status chan<- types.CompletionStatus) (*types.CompletionMessage, error) {
	c.modelsLock.Lock()
	provider, ok := c.modelToProvider[messageRequest.Model]
	c.modelsLock.Unlock()

	if !ok {
		return nil, fmt.Errorf("failed to find remote model %s", messageRequest.Model)
	}

	client, err := c.load(ctx, provider)
	if err != nil {
		return nil, err
	}

	toolName, modelName := types.SplitToolRef(messageRequest.Model)
	if modelName == "" {
		// modelName is empty, then the messageRequest.Model is not of the form 'modelName from provider'
		// Therefore, the modelName is the toolName
		modelName = toolName
	}
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

func (c *Client) parseModel(modelString string) (modelName, providerName string) {
	toolName, subTool := types.SplitToolRef(modelString)
	if subTool == "" {
		// This is just a plain model string "gpt4o"
		return toolName, c.defaultProvider
	}
	// This is a provider string "modelName from provider"
	return subTool, toolName
}

func (c *Client) Supports(ctx context.Context, modelString string) (bool, error) {
	_, providerName := c.parseModel(modelString)
	if providerName == "" {
		return false, nil
	}

	_, err := c.load(ctx, providerName)
	if err != nil {
		return false, err
	}

	c.modelsLock.Lock()
	defer c.modelsLock.Unlock()

	if c.modelToProvider == nil {
		c.modelToProvider = map[string]string{}
	}

	c.modelToProvider[modelString] = providerName
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
	if isHTTPURL(toolName) {
		remoteClient, err := c.clientFromURL(ctx, toolName)
		if err != nil {
			return nil, err
		}
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

	client, err := openai.NewClient(ctx, c.credStore, openai.Options{
		BaseURL:  strings.TrimSuffix(url, "/") + "/v1",
		Cache:    c.cache,
		CacheKey: prg.EntryToolID,
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) retrieveAPIKey(ctx context.Context, env, url string) (string, error) {
	return prompt.GetModelProviderCredential(ctx, c.credStore, url, env, fmt.Sprintf("Please provide your API key for %s", url), append(gcontext.GetEnv(ctx), c.envs...))
}

func isLocalhost(url string) bool {
	return strings.HasPrefix(url, "http://localhost") || strings.HasPrefix(url, "http://127.0.0.1") ||
		strings.HasPrefix(url, "https://localhost") || strings.HasPrefix(url, "https://127.0.0.1")
}
