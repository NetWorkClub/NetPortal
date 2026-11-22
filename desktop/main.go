package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/pkg/browser"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	webMode := flag.Bool("web", false, "Serve UI on 127.0.0.1 and open in default browser")
	webAddr := flag.String("web-addr", "127.0.0.1:0", "Loopback bind address for --web mode")
	flag.Parse()

	if runtime.GOOS == "linux" {
		// Prevent blank screens and Wayland Error 71 under NVIDIA/Mesa DMA-BUF in WebKitGTK 4.1.
		_ = os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	}

	app, err := NewApp("", "")
	if err != nil {
		log.Fatalf("init app: %v", err)
	}

	if *webMode {
		ln, token, err := startLoopbackWebServer(app, assets, *webAddr)
		if err != nil {
			log.Fatalf("start --web server: %v", err)
		}
		defer ln.Close()

		url := fmt.Sprintf("http://%s/?token=%s", ln.Addr().String(), token)
		log.Printf("NetPortal Web UI listening at %s", url)
		_ = browser.OpenURL(url)

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		app.shutdown(app.ctx)
		return
	}

	err = wails.Run(&options.App{
		Title:     "NetPortal",
		Width:     1200,
		Height:    760,
		MinWidth:  390,
		MinHeight: 640,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 11, G: 15, B: 23, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []any{
			app,
		},
	})
	if err != nil {
		log.Fatalf("wails run: %v", err)
	}
}
