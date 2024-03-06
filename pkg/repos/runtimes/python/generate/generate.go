package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	assetsURL        = "https://api.github.com/repos/indygreg/python-build-standalone/releases/%d/assets?per_page=1000"
	latestReleaseURL = "https://raw.githubusercontent.com/indygreg/python-build-standalone/latest-release/latest-release.json"
)

var (
	versions = []string{
		"3.10",
		"3.11",
		"3.12",
	}
	osKeyMap = map[osKey]string{
		{
			OS:   "linux",
			Arch: "amd64",
		}: "x86_64_v2-unknown-linux-gnu",
		{
			OS:   "linux",
			Arch: "arm64",
		}: "aarch64-unknown-linux-gnu",
		{
			OS:   "windows",
			Arch: "amd64",
		}: "x86_64-pc-windows-msvc-static",
		{
			OS:   "windows",
			Arch: "arm64",
		}: "aarch64-pc-windows-msvc-static",
		{
			OS:   "darwin",
			Arch: "amd64",
		}: "x86_64-apple-darwin",
		{
			OS:   "darwin",
			Arch: "arm64",
		}: "aarch64-apple-darwin",
	}
)

type osKey struct {
	OS   string
	Arch string
}

type Release struct {
	OS      string `json:"os,omitempty"`
	Arch    string `json:"arch,omitempty"`
	Version string `json:"version,omitempty"`
	URL     string `json:"url,omitempty"`
	Digest  string `json:"digest,omitempty"`
}

type latestRelease struct {
	Version        int    `json:"version"`
	Tag            string `json:"tag"`
	ReleaseURL     string `json:"release_url"`
	AssetURLPrefix string `json:"asset_url_prefix"`
}

type release struct {
	ID int `json:"id"`
}

type asset struct {
	URL                string `json:"url"`
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func main() {
	var output string
	flag.StringVar(&output, "output", "", "File to write to")
	flag.Parse()

	if err := mainErr(output); err != nil {
		log.Fatal(err)
	}
}

func toJSON(obj any, url string) error {
	url = strings.TrimPrefix(url, "<")

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	for _, link := range strings.Split(resp.Header.Get("link"), ", <") {
		nextURL, rel, _ := strings.Cut(link, ">; ")
		if rel == `rel="next"` {
			var more json.RawMessage
			if err := toJSON(&more, nextURL); err != nil {
				return err
			}
			// concatenate two json arrays in a fabulously hacky but totally valid approach
			data = append(data[:len(data)-1], []byte(",")...)
			data = append(data, more[1:]...)
		}
	}

	return json.Unmarshal(data, obj)
}

func getData(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	shaData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(shaData)), nil
}

func mainErr(output string) error {
	var (
		current  latestRelease
		release  release
		assets   []asset
		releases []Release
	)

	if err := toJSON(&current, latestReleaseURL); err != nil {
		return err
	}

	if err := toJSON(&release, current.ReleaseURL); err != nil {
		return err
	}

	if err := toJSON(&assets, fmt.Sprintf(assetsURL, release.ID)); err != nil {
		return err
	}

	for _, asset := range assets {
		if !strings.HasSuffix(asset.Name, "tar.gz") {
			continue
		}
		for _, version := range versions {
			ver, suffix, _ := strings.Cut(asset.Name, "+"+current.Tag+"-")
			prefix := fmt.Sprintf("cpython-%s.", version)
			if !strings.HasPrefix(ver, prefix) {
				continue
			}

			for osKey, spec := range osKeyMap {
				if suffix == spec+"-install_only.tar.gz" {
					digest, err := getData(asset.BrowserDownloadURL + ".sha256")
					if err != nil {
						return err
					}
					releases = append(releases, Release{
						OS:      osKey.OS,
						Arch:    osKey.Arch,
						Version: version,
						URL:     asset.BrowserDownloadURL,
						Digest:  digest,
					})
				}
			}
		}
	}

	f, err := os.Create(output)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(releases)
}
