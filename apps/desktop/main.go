package main

import (
	"embed"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	configRoot := os.Getenv("MFH_DESKTOP_CONFIG_DIR")
	if confirmation := os.Getenv("MFH_DESKTOP_RESET_CONFIRM"); confirmation != "" {
		if err := ResetSettings(configRoot, confirmation); err != nil {
			log.Fatal(err)
		}
	}
	app, err := NewApp(configRoot)
	if err != nil {
		log.Fatal(err)
	}
	if err := wails.Run(&options.App{
		Title: "MyFlowHub", Width: 1280, Height: 820, MinWidth: 980, MinHeight: 640,
		AssetServer: &assetserver.Options{Assets: assets},
		OnStartup:   app.Startup, OnShutdown: app.Shutdown, Bind: []interface{}{app},
		DragAndDrop: &options.DragAndDrop{EnableFileDrop: true},
		Windows:     &windows.Options{Theme: windows.SystemDefault},
	}); err != nil {
		log.Fatal(err)
	}
}
