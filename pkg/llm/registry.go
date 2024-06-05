package llm

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/gptscript-ai/gptscript/pkg/openai"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

type Client interface {
	Call(ctx context.Context, messageRequest types.CompletionRequest, status chan<- types.CompletionStatus) (*types.CompletionMessage, error)
	ListModels(ctx context.Context, providers ...string) (result []string, _ error)
	Supports(ctx context.Context, modelName string) (bool, error)
}

type Registry struct {
	clients []Client
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) AddClient(client Client) error {
	r.clients = append(r.clients, client)
	return nil
}

func (r *Registry) ListModels(ctx context.Context, providers ...string) (result []string, _ error) {
	for _, v := range r.clients {
		models, err := v.ListModels(ctx, providers...)
		if err != nil {
			return nil, err
		}
		result = append(result, models...)
	}
	sort.Strings(result)
	return result, nil
}

func (r *Registry) Call(ctx context.Context, messageRequest types.CompletionRequest, status chan<- types.CompletionStatus) (*types.CompletionMessage, error) {
	if messageRequest.Model == "" {
		return nil, fmt.Errorf("model is required")
	}

	var errs []error
	var oaiClient *openai.Client
	for _, client := range r.clients {
		ok, err := client.Supports(ctx, messageRequest.Model)
		if err != nil {
			// If we got an OpenAI invalid auth error back, store the OpenAI client for later.
			if errors.Is(err, openai.InvalidAuthError{}) {
				oaiClient = client.(*openai.Client)
			}

			errs = append(errs, err)
		} else if ok {
			return client.Call(ctx, messageRequest, status)
		}
	}

	if len(errs) > 0 && oaiClient != nil {
		// Prompt the user to enter their OpenAI API key and try again.
		if err := oaiClient.RetrieveAPIKey(ctx); err != nil {
			return nil, err
		}
		ok, err := oaiClient.Supports(ctx, messageRequest.Model)
		if err != nil {
			return nil, err
		} else if ok {
			return oaiClient.Call(ctx, messageRequest, status)
		}
	}

	if len(errs) == 0 {
		return nil, fmt.Errorf("failed to find a model provider for model [%s]", messageRequest.Model)
	}
	return nil, errors.Join(errs...)
}
