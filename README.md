# esbuild-plugin-importmaps

Esbuild plugin for utilizing import maps and url imports during bundling.
This plugin is for cases where you want to use esbuild from your go project.

## Usage

```go
package main

import (
	"os"
	
    "github.com/evanw/esbuild/pkg/api"
	"github.com/pushrbx/esbuild-plugin-importmap"
	"github.com/pushrbx/esbuild-plugin-importmap/importmap"
	
)

func main() {
	myImportMapData := importmap.Data{
		Imports: importmap.Imports{
			"preact-progressive-enhancement": "https://esm.sh/preact-progressive-enhancement@1.0.5",
        },
    }

	m := importmap.New(importmap.WithMap(myImportMapData))

	result := api.Build(api.BuildOptions{
		EntryPoints: []string{"input.js"},
		Outfile:     "output.js",
		Bundle:      true,
		Write:       true,
		LogLevel:    api.LogLevelInfo,
		Plugins: []api.Plugin{
			esbuild_plugin_importmap.NewPlugin(m),
		},
	})

	if len(result.Errors) > 0 {
		os.Exit(1)
	}
}
```

It's also possible to load an importmap file:

```go
package main

import (
	"os"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/pushrbx/esbuild-plugin-importmap"
	"github.com/pushrbx/esbuild-plugin-importmap/importmap"

)

func main() {
	m, err := importmap.LoadFromFile("./importmap.json")
	if err != nil {
		panic(err)
	}

	result := api.Build(api.BuildOptions{
		EntryPoints: []string{"input.js"},
		Outfile:     "output.js",
		Bundle:      true,
		Write:       true,
		LogLevel:    api.LogLevelInfo,
		Plugins: []api.Plugin{
			esbuild_plugin_importmap.NewPlugin(m),
		},
	})

	if len(result.Errors) > 0 {
		os.Exit(1)
	}
}
```
