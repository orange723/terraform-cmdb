package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"terraform-cmdb/internal/inventory"
	"terraform-cmdb/internal/server"
	"terraform-cmdb/internal/statefiles"
	"terraform-cmdb/internal/web"
)

func main() {
	const stateDir = "states"
	if len(os.Args) > 1 && os.Args[1] == "export" {
		if err := exportStatic(stateDir, "examples", "dist"); err != nil {
			log.Fatalf("export static site: %v", err)
		}
		return
	}

	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		log.Fatalf("create state dir: %v", err)
	}

	store := inventory.NewStore()
	app := server.New(store, server.Config{
		AppName:  "terraform-cmdb",
		StateDir: stateDir,
	})
	app.LoadStateDirectory()

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = "127.0.0.1:3000"
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		if err := app.App().Shutdown(); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()

	log.Printf("terraform-cmdb listening on http://%s", listenAddr)
	if err := app.App().Listen(listenAddr); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func exportStatic(stateDir, fallbackDir, outputDir string) error {
	sourceDir := stateDir
	result := statefiles.LoadDirectory(sourceDir)
	if len(result.Snapshot.Machines) == 0 && result.Snapshot.LastError == "" {
		sourceDir = fallbackDir
		result = statefiles.LoadDirectory(sourceDir)
	}
	snapshot := result.Snapshot

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	index := web.RenderIndex(web.IndexData{
		FileName:     snapshot.FileName,
		SourceFiles:  snapshot.SourceFiles,
		StateDir:     sourceDir,
		Terraform:    snapshot.Terraform,
		Machines:     snapshot.Machines,
		LastError:    snapshot.LastError,
		RawResources: snapshot.RawResources,
		Static:       true,
	})
	if err := os.WriteFile(filepath.Join(outputDir, "index.html"), []byte(index), 0o644); err != nil {
		return err
	}

	instances, err := json.MarshalIndent(inventory.InstancesPayload(snapshot), "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outputDir, "instances.json"), append(instances, '\n'), 0o644); err != nil {
		return err
	}

	log.Printf("exported %d machines to %s", len(snapshot.Machines), outputDir)
	return nil
}
