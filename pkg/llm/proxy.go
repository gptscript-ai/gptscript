package llm

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/builtin"
	"github.com/gptscript-ai/gptscript/pkg/openai"
)

func (r *Registry) ProxyInfo() (string, string, error) {
	r.proxyLock.Lock()
	defer r.proxyLock.Unlock()

	if r.proxyURL != "" {
		return r.proxyToken, r.proxyURL, nil
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", "", err
	}

	go func() {
		_ = http.Serve(l, r)
		r.proxyLock.Lock()
		defer r.proxyLock.Unlock()
		_ = l.Close()
		r.proxyURL = ""
	}()

	r.proxyURL = "http://" + l.Addr().String()
	return r.proxyToken, r.proxyURL, nil
}

func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.proxyToken != strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ") {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	inBytes, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var (
		model string
		data  map[string]any
	)

	if json.Unmarshal(inBytes, &data) == nil {
		model, _ = data["model"].(string)
	}

	if model == "" {
		model = builtin.GetDefaultModel()
	}

	c, err := r.getClient(req.Context(), model, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	oai, ok := c.(*openai.Client)
	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	auth, targetURL := oai.ProxyInfo()
	if targetURL == "" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	newURL, err := url.Parse(targetURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newURL.Path = path.Join(newURL.Path, req.URL.Path)

	rp := httputil.ReverseProxy{
		Director: func(proxyReq *http.Request) {
			proxyReq.Body = io.NopCloser(bytes.NewReader(inBytes))
			proxyReq.URL = newURL
			proxyReq.Header.Del("Authorization")
			proxyReq.Header.Add("Authorization", "Bearer "+auth)
			proxyReq.Host = newURL.Hostname()
		},
	}
	rp.ServeHTTP(w, req)
}
