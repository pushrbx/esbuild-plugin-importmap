package esbuild_plugin_importmap

import (
	"github.com/evanw/esbuild/pkg/api"
	"github.com/pushrbx/esbuild-plugin-importmap/importmap"
	"os"
	"testing"
)

func getFileTreePlugin(t *testing.T, staticTestContent string) api.Plugin {
	t.Helper()

	return api.Plugin{
		Name: "file-tree",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{
				Filter: `^\..*$`,
			}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				t.Logf("resolving: %s -- %s", args.Path, args.Importer)

				return api.OnResolveResult{
					Path: args.Path, Namespace: "file-tree",
				}, nil
			})

			build.OnLoad(api.OnLoadOptions{
				Filter:    ".*",
				Namespace: "file-tree",
			}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				t.Logf("loading: %s", args.Path)

				if args.Path == "./index.js" {
					return api.OnLoadResult{
						Contents: &staticTestContent,
						Loader:   api.LoaderJS,
					}, nil
				}

				fileContentsRaw, err := os.ReadFile(args.Path)

				if err != nil {
					return api.OnLoadResult{}, err
				}

				fileContents := string(fileContentsRaw)

				return api.OnLoadResult{
					Contents: &fileContents,
					Loader:   api.LoaderJS,
				}, nil
			})
		},
	}
}

func TestPluginWithRemoteModules(t *testing.T) {
	fileTreePlugin := getFileTreePlugin(t, "import {define} from 'preact-progressive-enhancement'; console.log(define);")
	plugin, err := NewPlugin(WithMap(importmap.Data{
		Imports: importmap.Imports{
			"preact-progressive-enhancement": "https://esm.sh/preact-progressive-enhancement@1.0.5",
		},
	}))
	if err != nil {
		t.Fatal(err)
	}

	buildOptions := api.BuildOptions{
		Bundle:            true,
		MinifyIdentifiers: false,
		MinifySyntax:      false,
		MinifyWhitespace:  false,
		Format:            api.FormatESModule,
		LogLevel:          api.LogLevelInfo,
		Write:             false,
		EntryPoints:       []string{"./index.js"},
		Plugins: []api.Plugin{
			fileTreePlugin,
			plugin,
		},
	}

	result := api.Build(buildOptions)

	if len(result.Errors) > 0 {
		t.Error("failed to build")
		t.FailNow()
	}

	if len(result.OutputFiles) != 1 {
		t.Errorf("expected 1 output file, got %d", len(result.OutputFiles))
		t.FailNow()
	}
}

func TestPluginWithLocalModules(t *testing.T) {
	fileTreePlugin := getFileTreePlugin(t, "import {define} from '@/testModule.js'; import {dummy} from '@/testfolder/testfile.js'; console.log(define); console.log(dummy);")
	plugin, err := NewPlugin(WithMap(importmap.Data{
		Imports: importmap.Imports{
			"@/": "./",
		},
	}))
	if err != nil {
		t.Fatal(err)
	}

	buildOptions := api.BuildOptions{
		Bundle:            true,
		MinifyIdentifiers: false,
		MinifySyntax:      false,
		MinifyWhitespace:  false,
		Format:            api.FormatESModule,
		LogLevel:          api.LogLevelInfo,
		Write:             false,
		EntryPoints:       []string{"./index.js"},
		Plugins: []api.Plugin{
			fileTreePlugin,
			plugin,
		},
	}

	result := api.Build(buildOptions)

	if len(result.Errors) > 0 {
		t.Error("failed to build")
		t.FailNow()
	}

	if len(result.OutputFiles) != 1 {
		t.Errorf("expected 1 output file, got %d", len(result.OutputFiles))
		t.FailNow()
	}

	t.Logf("Result contents:\n%s", result.OutputFiles[0].Contents)
}
