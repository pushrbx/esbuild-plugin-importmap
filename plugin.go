package esbuild_plugin_importmap

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/pushrbx/esbuild-plugin-importmap/importmap"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const namespace = "importmap-url"

// Config is the configuration object for the plugin
type Config struct {
	ImportMapData *importmap.Data
	ImportMap     importmap.IImportMap
}

type Option func(config *Config)

// NewPlugin creates a new import map esbuild plugin
func NewPlugin(opts ...Option) (api.Plugin, error) {
	config := &Config{}

	for _, opt := range opts {
		opt(config)
	}

	var importMap importmap.IImportMap
	if config.ImportMapData != nil {
		var err error
		importMap, err = importmap.New(
			importmap.WithMap(*config.ImportMapData),
		)

		if err != nil {
			return api.Plugin{}, err
		}
	}
	if config.ImportMap != nil {
		importMap = config.ImportMap
	}
	if importMap == nil {
		return api.Plugin{}, fmt.Errorf("no importmap was provided")
	}
	return api.Plugin{
		Name:  "importmap-url",
		Setup: setup(importMap),
	}, nil
}

// WithMap sets the import map data
func WithMap(importMap importmap.Data) Option {
	return func(config *Config) {
		config.ImportMapData = &importMap
	}
}

// WithImportMapPath sets the path to the import map json file
func WithImportMapPath(path string) Option {
	return func(config *Config) {
		data, err := importmap.LoadFromFile(path)
		if err != nil {
			panic(err)
		}
		config.ImportMap = data
	}
}

func setup(importMap importmap.IImportMap) func(b api.PluginBuild) {
	return func(b api.PluginBuild) {
		b.OnResolve(api.OnResolveOptions{
			Filter: "^[^.].*$",
		}, onResolve(importMap))

		b.OnLoad(api.OnLoadOptions{
			Filter:    ".*",
			Namespace: namespace,
		}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
			loader := api.LoaderJS
			ext := path.Ext(args.Path)
			switch ext {
			case ".ts":
				loader = api.LoaderTS
				break
			case ".tsx":
				loader = api.LoaderTSX
			case ".jsx":
				loader = api.LoaderJSX
			}
			if !strings.Contains(args.Path, "http") {
				cleanedPath := strings.Replace(args.Path, "file://", "", 1)
				if filepath.IsLocal(cleanedPath) || filepath.IsAbs(cleanedPath) {
					fileContents, err := os.ReadFile(cleanedPath)
					if err != nil {
						return api.OnLoadResult{}, err
					}

					fileContentsStr := string(fileContents)

					return api.OnLoadResult{
						Contents: &fileContentsStr,
						Loader:   loader,
					}, nil
				} else {
					return api.OnLoadResult{}, errors.New("invalid path: " + args.Path)
				}
			} else {
				const maxRetries = 3
				const baseDelay = 300 * time.Millisecond
				var statusCodesToRetry = []int{429, 500, 502, 503, 504}

				for attempt := 0; attempt < maxRetries; attempt++ {
					// download from url
					resp, err := http.Get(args.Path)

					if err != nil {
						return api.OnLoadResult{}, err
					}

					var buf bytes.Buffer

					_, err = io.Copy(&buf, resp.Body)
					if err != nil {
						return api.OnLoadResult{}, err
					}

					contents := buf.String()

					_ = resp.Body.Close()

					if strings.Contains(contents, "SlowDown") || slices.Index(statusCodesToRetry, resp.StatusCode) > -1 {
						if attempt < maxRetries-1 {
							delay := time.Duration(1<<uint(attempt)) * baseDelay
							time.Sleep(delay)
							continue
						} else {
							return api.OnLoadResult{}, fmt.Errorf("failed to download file after %d attempts, received %d status code: %s", maxRetries, resp.StatusCode, args.Path)
						}
					}

					return api.OnLoadResult{
						Contents: &contents,
						Loader:   loader,
					}, nil
				}

				return api.OnLoadResult{}, fmt.Errorf("failed to download file after %d attempts: %s", maxRetries, args.Path)
			}
		})
	}
}

func onResolve(importMap importmap.IImportMap) func(args api.OnResolveArgs) (api.OnResolveResult, error) {
	return func(args api.OnResolveArgs) (api.OnResolveResult, error) {
		parsedImporterUrl, err := url.Parse(args.Importer)
		if err != nil {
			return api.OnResolveResult{}, err
		}

		resolvedPath, err := importMap.ResolveWithParent(args.Path, parsedImporterUrl)
		if err != nil {
			return api.OnResolveResult{}, err
		}
		// this should call our custom importmap object
		return api.OnResolveResult{
			Path:      resolvedPath,
			Namespace: "importmap-url",
		}, nil
	}
}
