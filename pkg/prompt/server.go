package prompt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

func NewServer(ctx context.Context, envs []string) ([]string, error) {
	for _, env := range envs {
		_, v, ok := strings.Cut(env, types.PromptTokenEnvVar+"=")
		if ok && v != "" {
			return nil, nil
		}
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	token := uuid.NewString()
	s := http.Server{
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+token {
				rw.WriteHeader(http.StatusUnauthorized)
				_, _ = rw.Write([]byte("Unauthorized"))
				return
			}

			var req types.Prompt
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				_, _ = rw.Write([]byte(err.Error()))
				return
			}

			resp, err := sysPrompt(r.Context(), req)
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				_, _ = rw.Write([]byte(err.Error()))
				return
			}

			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte(resp))
		}),
	}

	context.AfterFunc(ctx, func() {
		_ = s.Shutdown(context.Background())
	})

	go func() {
		if err := s.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed to run prompt server: %v", err)
		}
	}()

	return []string{
		fmt.Sprintf("%s=http://%s", types.PromptURLEnvVar, l.Addr().String()),
		fmt.Sprintf("%s=%s", types.PromptTokenEnvVar, token),
	}, nil
}
