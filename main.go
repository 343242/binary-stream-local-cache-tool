package main

import (
	"fastReadFile/desktop/backend"
	"io/fs"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

func main() {
	app := backend.NewApp()
	if err := wails.Run(&options.App{
		Title:       "Binary Stream Cache Tool",
		Width:       1440,
		Height:      960,
		MinWidth:    1280,
		MinHeight:   800,
		OnStartup:   app.Startup,
		Bind:        []any{app},
		AssetServer: &assetserver.Options{Assets: mustLoadAssets()},
	}); err != nil {
		panic(err)
	}
}

func mustLoadAssets() fs.FS {
	assets := os.DirFS("desktop/frontend/dist")
	if _, err := fs.Stat(assets, "."); err == nil {
		return assets
	}
	return os.DirFS(".")
}
