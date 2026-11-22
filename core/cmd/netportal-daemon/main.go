// Command netportal-daemon runs the privileged NetPortal background service
// managing TUN interfaces, protocol engines, and system routing cleanup.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NetWorkClub/NetPortal/core/daemon"
	"github.com/NetWorkClub/NetPortal/core/engine"
	"github.com/NetWorkClub/NetPortal/core/engine/awg"
	"github.com/NetWorkClub/NetPortal/core/engine/byedpi"
	"github.com/NetWorkClub/NetPortal/core/engine/psiphon"
	"github.com/NetWorkClub/NetPortal/core/engine/singbox"
	"github.com/NetWorkClub/NetPortal/core/engine/smartdns"
	"github.com/NetWorkClub/NetPortal/core/engine/zapret"
)

func main() {
	endpoint := flag.String("endpoint", daemon.DefaultEndpoint(), "IPC pipe or unix socket path")
	flag.Parse()

	reg := engine.NewRegistry()
	for _, eng := range []engine.Engine{
		awg.New(),
		singbox.New(),
		psiphon.New(),
		byedpi.New(),
		zapret.New(),
		smartdns.New(),
	} {
		if err := reg.Register(eng); err != nil {
			log.Fatalf("register %s engine: %v", eng.ID(), err)
		}
	}

	ln, err := daemon.ListenIPC(*endpoint)
	if err != nil {
		log.Fatalf("listen ipc %s: %v", *endpoint, err)
	}
	defer ln.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	srv := daemon.NewServer(reg, cancel)
	log.Printf("netportal-daemon listening on %s", *endpoint)

	if err := srv.Serve(ctx, ln); err != nil {
		log.Printf("daemon serve error: %v", err)
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()
	if err := srv.StopActive(stopCtx); err != nil {
		log.Printf("stop active engine error: %v", err)
	}
}
