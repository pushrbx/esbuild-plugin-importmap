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
	"path/filepath"
	"strings"
)

const namespace = "importmap-url"

type Config struct {
	ImportMapData *importmap.Data
	ImportMap     importmap.IImportMap
}

type Option func(config *Config)

// NewPlugin creates a new importmap esbuild plugin
func NewPlugin(opts ...Option) (*api.Plugin, error) {
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
			return nil, err
		}
	}
	if config.ImportMap != nil {
		importMap = config.ImportMap
	}
	if importMap == nil {
		return nil, fmt.Errorf("no importmap was provided")
	}
	return &api.Plugin{
		Name:  "importmap-url",
		Setup: setup(importMap),
	}, nil
}

func WithMap(importMap importmap.Data) Option {
	return func(config *Config) {
		config.ImportMapData = &importMap
	}
}

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
			loader := api.LoaderJS | api.LoaderTS | api.LoaderJSX | api.LoaderTSX
			if !strings.Contains(args.Path, "http") {
				if filepath.IsLocal(args.Path) || filepath.IsAbs(args.Path) {
					fileContents, err := os.ReadFile(args.Path)
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
				// download from url
				resp, err := http.Get(args.Path)

				if err != nil {
					return api.OnLoadResult{}, err
				}

				defer func(Body io.ReadCloser) {
					_ = Body.Close()
				}(resp.Body)

				var buf bytes.Buffer

				_, err = io.Copy(&buf, resp.Body)
				if err != nil {
					return api.OnLoadResult{}, err
				}

				contents := buf.String()

				return api.OnLoadResult{
					Contents: &contents,
					Loader:   loader,
				}, nil
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
