package builtin

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	// These lists come from https://api.search.brave.com/app/documentation/web-search/codes, but you must be logged in in order to view the docs.
	braveSupportedCountries = sets.New("AR", "AU", "AT", "BE", "BR", "CA", "CL", "DK", "FI", "FR", "DE", "HK", "IN", "ID", "IT", "JP", "KR", "MY", "MX", "NL", "NZ", "NO", "CN", "PL", "PT", "PH", "RU", "SA", "ZA", "ES", "SE", "CH", "TW", "TR", "GB", "US")
	braveSupportedLanguages = sets.New("ar", "eu", "bn", "bg", "ca", "zh-hans", "zh-hant", "hr", "cs", "da", "nl", "en", "en-gb", "et", "fi", "fr", "gl", "de", "gu", "he", "hi", "hu", "is", "it", "jp", "kn", "ko", "lv", "lt", "ms", "ml", "mr", "nb", "pl", "pt-br", "pt-pt", "pa", "ro", "ru", "sr", "sk", "sl", "es", "sv", "ta", "te", "th", "tr", "uk", "vi")
)

func SysSearchBrave(ctx context.Context, env []string, input string) (string, error) {
	token := os.Getenv("GPTSCRIPT_BRAVE_SEARCH_TOKEN")
	if token == "" {
		return "", fmt.Errorf("GPTSCRIPT_BRAVE_SEARCH_TOKEN is not set")
	}

	var params struct {
		Query      string `json:"q"`
		Country    string `json:"country"`
		SearchLang string `json:"search_lang"`
		Offset     string `json:"offset"`
	}

	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", err
	}

	// Validate parameters
	if params.Country != "" && !braveSupportedCountries.Has(params.Country) {
		return "", fmt.Errorf("unsupported country: %s", params.Country)
	}
	if params.SearchLang != "" && !braveSupportedLanguages.Has(params.SearchLang) {
		return "", fmt.Errorf("unsupported language: %s", params.SearchLang)
	}

	baseURL := "https://api.search.brave.com/res/v1/web/search"
	queryParams := url.Values{}
	queryParams.Add("q", params.Query)

	if params.Country != "" {
		queryParams.Add("country", params.Country)
	}
	if params.SearchLang != "" {
		queryParams.Add("search_lang", params.SearchLang)
	}
	if params.Offset != "" {
		queryParams.Add("offset", params.Offset)
	}

	fullURL := fmt.Sprintf("%s?%s", baseURL, queryParams.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("X-Subscription-Token", os.Getenv("GPTSCRIPT_BRAVE_SEARCH_TOKEN"))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer func() {
		_ = res.Body.Close()
	}()

	gzipReader, err := gzip.NewReader(res.Body)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(gzipReader)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
