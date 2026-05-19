package main

import (
	"log"
	"os"

	"terraform-cmdb/internal/inventory"
	"terraform-cmdb/internal/server"
)

func main() {
	const stateDir = "states"

	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		log.Fatalf("create state dir: %v", err)
	}

	store := inventory.NewStore()
	app := server.New(store, server.Config{
		AppName:  "terraform-cmdb",
		StateDir: stateDir,
	})
	app.LoadStateDirectory()

	log.Println("terraform-cmdb listening on http://127.0.0.1:3000")
	log.Fatal(app.App().Listen(":3000"))
}
