// Package main is the entrypoint for the PersonalSecureEncrypter Wails application.
// It configures the Wails runtime, embeds the compiled frontend assets, and starts
// the desktop window with a dark translucent background.
package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

// Embed the entire compiled frontend (Vite output) into the binary.
// This directive requires `frontend/dist/` to exist at build time —
// the Wails CLI handles building the frontend before compiling Go.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "Reg-X",
		Width:     1100,
		Height:    720,
		MinWidth:  900,
		MinHeight: 600,

		// Embed compiled React build as the application's asset server.
		AssetServer: &assetserver.Options{
			Assets: assets,
		},

		// Dark background matching the frontend body color (#0f0f1a).
		BackgroundColour: &options.RGBA{R: 15, G: 15, B: 26, A: 255},

		// Lifecycle callbacks — the App struct receives the Wails context.
		OnStartup:  app.startup,
		OnDomReady: app.domReady,

		// Bind the App struct so all its exported methods become callable
		// from the frontend via auto-generated TypeScript wrappers.
		Bind: []interface{}{
			app,
		},

		// Enable native file drag-and-drop from the OS file manager.
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: false,
			CSSDropProperty:    "--wails-drop-target",
			CSSDropValue:       "drop",
		},

		// Windows-specific: enable Mica backdrop for a premium translucent look.
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			BackdropType:         windows.Mica,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
