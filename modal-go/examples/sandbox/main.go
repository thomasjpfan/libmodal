package main

import (
	"context"
	"io"
	"log"

	"github.com/modal-labs/libmodal/modal-go"
)

func main() {
	ctx := context.Background()

	app, err := modal.AppLookup(ctx, "libmodal-example", &modal.LookupOptions{CreateIfMissing: true})
	if err != nil {
		log.Fatalf("Failed to lookup or create app: %v", err)
	}

	image := modal.NewImageFromRegistry("alpine:3.21", nil)

	sb, err := app.CreateSandbox(image, &modal.SandboxOptions{
		Command: []string{"cat"},
	})
	if err != nil {
		log.Fatalf("Failed to create sandbox: %v", err)
	}
	log.Printf("sandbox: %s\n", sb.SandboxId)

	sbFromId, err := modal.SandboxFromId(ctx, sb.SandboxId)
	if err != nil {
		log.Fatalf("Failed to get sandbox with Id: %v", err)
	}
	log.Printf("Queried sandbox with id: %v", sbFromId.SandboxId)

	_, err = sb.Stdin.Write([]byte("this is input that should be mirrored by cat"))
	if err != nil {
		log.Fatalf("Failed to write to sandbox stdin: %v", err)
	}
	err = sb.Stdin.Close()
	if err != nil {
		log.Fatalf("Failed to close sandbox stdin: %v", err)
	}

	output, err := io.ReadAll(sb.Stdout)
	if err != nil {
		log.Fatalf("Failed to read from sandbox stdout: %v", err)
	}

	log.Printf("output: %s\n", string(output))
}
