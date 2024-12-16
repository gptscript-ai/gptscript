package engine

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/types"
)

const DaemonURLSuffix = ".daemon.gptscript.local"

func (e *Engine) runHTTP(ctx context.Context, prg *types.Program, tool types.Tool, input string) (cmdRet *Return, cmdErr error) {
	envMap := map[string]string{}

	for _, env := range appendInputAsEnv(nil, input) {
		k, v, _ := strings.Cut(env, "=")
		envMap[k] = v
	}

	for _, env := range e.Env {
		k, v, _ := strings.Cut(env, "=")
		envMap[k] = v
	}

	toolURL := strings.Split(tool.Instructions, "\n")[0][2:]
	toolURL = os.Expand(toolURL, func(s string) string {
		return url.PathEscape(envMap[s])
	})

	parsed, err := url.Parse(toolURL)
	if err != nil {
		return nil, err
	}

	var tlsConfigForDaemonRequest *tls.Config
	if strings.HasSuffix(parsed.Hostname(), DaemonURLSuffix) {
		referencedToolName := strings.TrimSuffix(parsed.Hostname(), DaemonURLSuffix)
		referencedToolRefs, ok := tool.ToolMapping[referencedToolName]
		if !ok || len(referencedToolRefs) != 1 {
			return nil, fmt.Errorf("invalid reference [%s] to tool [%s] from [%s], missing \"tools: %s\" parameter", toolURL, referencedToolName, tool.Source, referencedToolName)
		}
		referencedTool, ok := prg.ToolSet[referencedToolRefs[0].ToolID]
		if !ok {
			return nil, fmt.Errorf("failed to find tool [%s] for [%s]", referencedToolName, parsed.Hostname())
		}
		toolURL, err = e.startDaemon(referencedTool)
		if err != nil {
			return nil, err
		}
		toolURLParsed, err := url.Parse(toolURL)
		if err != nil {
			return nil, err
		}
		parsed.Host = toolURLParsed.Host
		toolURL = parsed.String()

		// Find the certificate corresponding to this daemon tool
		certificates.lock.Lock()
		daemonCert, exists := certificates.daemonCerts[referencedTool.ID]
		clientCert := certificates.clientCert
		certificates.lock.Unlock()

		if !exists {
			return nil, fmt.Errorf("missing daemon certificate for [%s]", referencedTool.ID)
		}

		// Create a pool for the certificate to treat as a CA
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(daemonCert.Cert) {
			return nil, fmt.Errorf("failed to append daemon certificate for [%s]", referencedTool.ID)
		}

		tlsClientCert, err := tls.X509KeyPair(clientCert.Cert, clientCert.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to create client certificate: %v", err)
		}

		// Create TLS config for use in the HTTP client later
		tlsConfigForDaemonRequest = &tls.Config{
			Certificates:       []tls.Certificate{tlsClientCert},
			RootCAs:            pool,
			InsecureSkipVerify: false,
		}
	}

	if tool.Blocking {
		return &Return{
			Result: &toolURL,
		}, nil
	}

	if body, ok := envMap["BODY"]; ok {
		input = body
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, toolURL, strings.NewReader(input))
	if err != nil {
		return nil, err
	}

	for _, k := range slices.Sorted(maps.Keys(envMap)) {
		if strings.HasPrefix(k, "GPTSCRIPT_WORKSPACE_") {
			req.Header.Add("X-GPTScript-Env", k+"="+envMap[k])
		}
	}

	for _, prefix := range strings.Split(envMap["GPTSCRIPT_HTTP_ENV_PREFIX"], ",") {
		if prefix == "" {
			continue
		}
		for _, k := range slices.Sorted(maps.Keys(envMap)) {
			if strings.HasPrefix(k, prefix) {
				req.Header.Add("X-GPTScript-Env", k+"="+envMap[k])
			}
		}
	}

	for _, k := range strings.Split(envMap["GPTSCRIPT_HTTP_ENV"], ",") {
		if k == "" {
			continue
		}
		v := envMap[k]
		if v != "" {
			req.Header.Add("X-GPTScript-Env", k+"="+v)
		}
	}

	req.Header.Set("X-GPTScript-Tool-Name", tool.Parameters.Name)

	if err := json.Unmarshal([]byte(input), &map[string]any{}); err == nil {
		req.Header.Set("Content-Type", "application/json")
	} else {
		req.Header.Set("Content-Type", "text/plain")
	}

	var httpClient *http.Client
	if tlsConfigForDaemonRequest != nil {
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfigForDaemonRequest,
			},
		}
	} else {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error in request to [%s] [%d]: %s: %s", toolURL, resp.StatusCode, resp.Status, body)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.Header.Get("Content-Type") == "application/json" && strings.HasPrefix(string(content), "\"") {
		// This is dumb hack when something returns a string in JSON format, just decode it to a string
		var s string
		if err := json.Unmarshal(content, &s); err == nil {
			return &Return{
				Result: &s,
			}, nil
		}
	}

	s := string(content)
	return &Return{
		Result: &s,
	}, nil
}
