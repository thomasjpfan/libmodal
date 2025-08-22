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

	image := modal.NewImageFromRegistry("python:3.13-slim", nil)

	sb, err := app.CreateSandbox(image, nil)
	if err != nil {
		log.Fatalf("Failed to create sandbox: %v", err)
	}
	log.Println("Started sandbox:", sb.SandboxId)
	defer sb.Terminate()

	p, err := sb.Exec(
		[]string{
			"python",
			"-c",
			`
import time
import sys
for i in range(50000):
	if i % 1000 == 0:
		time.sleep(0.01)
	print(i)
	print(i, file=sys.stderr)`,
		},
		modal.ExecOptions{
			Stdout: modal.Pipe,
			Stderr: modal.Pipe,
		},
	)
	if err != nil {
		log.Fatalf("Failed to execute command in sandbox: %v", err)
	}

	contentStdout, err := io.ReadAll(p.Stdout)
	if err != nil {
		log.Fatalf("Failed to read stdout: %v", err)
	}
	contentStderr, err := io.ReadAll(p.Stderr)
	if err != nil {
		log.Fatalf("Failed to read stderr: %v", err)
	}

	log.Printf("Got %d bytes stdout and %d bytes stderr\n", len(contentStdout), len(contentStderr))
	returnCode, err := p.Wait()
	if err != nil {
		log.Fatalf("Failed to wait for process completion: %v", err)
	}
	log.Println("Return code:", returnCode)

	secret, err := modal.SecretFromName(context.Background(), "libmodal-test-secret", &modal.SecretFromNameOptions{RequiredKeys: []string{"c"}})
	if err != nil {
		log.Fatalf("Unable to get secret: %v", err)
	}

	// Passing Secrets in a command
	p, err = sb.Exec([]string{"printenv", "c"}, modal.ExecOptions{Stdout: modal.Pipe, Stderr: modal.Pipe, Secrets: []*modal.Secret{secret}})
	if err != nil {
		log.Fatalf("Faield to execute env command in sandbox: %v", err)
	}

	secretStdout, err := io.ReadAll(p.Stdout)
	if err != nil {
		log.Fatalf("Failed to read stdout: %v", err)
	}
	log.Printf("Got environment variable c=%v", string(secretStdout))
}
