package main

import (
	"embed"
	"encoding/json"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/sjakovic/yearfold/internal/app"
	"github.com/sjakovic/yearfold/internal/assets"
)

//go:embed all:frontend/dist
var frontend embed.FS

//go:embed wails.json
var projectConfig []byte

func version() string {
	var cfg struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if err := json.Unmarshal(projectConfig, &cfg); err != nil || cfg.Info.ProductVersion == "" {
		return "dev"
	}
	return "v" + cfg.Info.ProductVersion
}

func main() {
	api := app.New(version())

	err := wails.Run(&options.App{
		Title:     "Yearfold",
		Width:     1400,
		Height:    900,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets:  frontend,
			Handler: assets.Handler(app.CurrentLibrary(api)),
		},
		BackgroundColour: &options.RGBA{R: 24, G: 26, B: 30, A: 1},
		OnStartup:        app.Startup(api),
		OnShutdown:       app.Shutdown(api),
		Bind:             []interface{}{api},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
