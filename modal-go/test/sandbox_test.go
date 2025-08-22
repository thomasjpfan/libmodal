//nolint:staticcheck // SA1019 We need to use deprecated API for testing
package test

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/modal-labs/libmodal/modal-go"
	"github.com/onsi/gomega"
)

func TestCreateOneSandbox(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	app, err := modal.AppLookup(context.Background(), "libmodal-test", &modal.LookupOptions{CreateIfMissing: true})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(app.Name).To(gomega.Equal("libmodal-test"))

	image, err := app.ImageFromRegistry("alpine:3.21", nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	sb, err := app.CreateSandbox(image, nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(sb.SandboxId).ShouldNot(gomega.BeEmpty())

	err = sb.Terminate()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	exitcode, err := sb.Wait()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(exitcode).To(gomega.Equal(137))
}

func TestPassCatToStdin(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	app, err := modal.AppLookup(context.Background(), "libmodal-test", &modal.LookupOptions{CreateIfMissing: true})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	image, err := app.ImageFromRegistry("alpine:3.21", nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Spawn a sandbox running the "cat" command.
	sb, err := app.CreateSandbox(image, &modal.SandboxOptions{Command: []string{"cat"}})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Write to the sandbox's stdin and read from its stdout.
	_, err = sb.Stdin.Write([]byte("this is input that should be mirrored by cat"))
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	err = sb.Stdin.Close()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	output, err := io.ReadAll(sb.Stdout)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(string(output)).To(gomega.Equal("this is input that should be mirrored by cat"))

	// Terminate the sandbox.
	err = sb.Terminate()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
}

func TestIgnoreLargeStdout(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	app, err := modal.AppLookup(context.Background(), "libmodal-test", &modal.LookupOptions{CreateIfMissing: true})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	image, err := app.ImageFromRegistry("python:3.13-alpine", nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	sb, err := app.CreateSandbox(image, nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer sb.Terminate()

	p, err := sb.Exec([]string{"python", "-c", `print("a" * 1_000_000)`}, modal.ExecOptions{Stdout: modal.Ignore})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	buf, err := io.ReadAll(p.Stdout)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(len(buf)).To(gomega.Equal(0)) // Stdout is ignored

	// Stdout should be consumed after cancel, without blocking the process.
	exitCode, err := p.Wait()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(exitCode).To(gomega.Equal(0))
}

func TestSandboxCreateOptions(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	app, err := modal.AppLookup(ctx, "libmodal-test", &modal.LookupOptions{
		CreateIfMissing: true,
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	image, err := app.ImageFromRegistry("alpine:3.21", nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	sb, err := app.CreateSandbox(image, &modal.SandboxOptions{
		Command: []string{"echo", "hello, params"},
		Cloud:   "aws",
		Regions: []string{"us-east-1", "us-west-2"},
		Verbose: true,
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(sb).ShouldNot(gomega.BeNil())
	g.Expect(sb.SandboxId).Should(gomega.HavePrefix("sb-"))

	defer sb.Terminate()

	exitCode, err := sb.Wait()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(exitCode).Should(gomega.Equal(0))

	_, err = app.CreateSandbox(image, &modal.SandboxOptions{
		Cloud: "invalid-cloud",
	})
	g.Expect(err).Should(gomega.HaveOccurred())
	g.Expect(err.Error()).Should(gomega.ContainSubstring("InvalidArgument"))

	_, err = app.CreateSandbox(image, &modal.SandboxOptions{
		Regions: []string{"invalid-region"},
	})
	g.Expect(err).Should(gomega.HaveOccurred())
	g.Expect(err.Error()).Should(gomega.ContainSubstring("InvalidArgument"))
}

func TestSandboxExecOptions(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	app, err := modal.AppLookup(context.Background(), "libmodal-test", &modal.LookupOptions{CreateIfMissing: true})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	image, err := app.ImageFromRegistry("alpine:3.21", nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	sb, err := app.CreateSandbox(image, nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer sb.Terminate()

	// Test with a custom working directory and timeout.
	p, err := sb.Exec([]string{"pwd"}, modal.ExecOptions{
		Workdir: "/tmp",
		Timeout: 5,
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	output, err := io.ReadAll(p.Stdout)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(string(output)).To(gomega.Equal("/tmp\n"))

	exitCode, err := p.Wait()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(exitCode).To(gomega.Equal(0))
}

func TestSandboxWithVolume(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	app, err := modal.AppLookup(ctx, "libmodal-test", &modal.LookupOptions{
		CreateIfMissing: true,
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	image, err := app.ImageFromRegistry("alpine:3.21", nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	volume, err := modal.VolumeFromName(ctx, "libmodal-test-sandbox-volume", &modal.VolumeFromNameOptions{
		CreateIfMissing: true,
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	sandbox, err := app.CreateSandbox(image, &modal.SandboxOptions{
		Command: []string{"echo", "volume test"},
		Volumes: map[string]*modal.Volume{
			"/mnt/test": volume,
		},
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(sandbox).ShouldNot(gomega.BeNil())
	g.Expect(sandbox.SandboxId).Should(gomega.HavePrefix("sb-"))

	exitCode, err := sandbox.Wait()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(exitCode).Should(gomega.Equal(0))
}

func TestSandboxWithReadOnlyVolume(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	app, err := modal.AppLookup(ctx, "libmodal-test", &modal.LookupOptions{
		CreateIfMissing: true,
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	image := modal.NewImageFromRegistry("alpine:3.21", nil)

	volume, err := modal.VolumeFromName(ctx, "libmodal-test-sandbox-volume", &modal.VolumeFromNameOptions{
		CreateIfMissing: true,
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	readOnlyVolume := volume.ReadOnly()
	g.Expect(readOnlyVolume.IsReadOnly()).To(gomega.BeTrue())

	sb, err := app.CreateSandbox(image, &modal.SandboxOptions{
		Command: []string{"sh", "-c", "echo 'test' > /mnt/test/test.txt"},
		Volumes: map[string]*modal.Volume{
			"/mnt/test": readOnlyVolume,
		},
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	exitCode, err := sb.Wait()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(exitCode).Should(gomega.Equal(1))

	stderr, err := io.ReadAll(sb.Stderr)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(string(stderr)).Should(gomega.ContainSubstring("Read-only file system"))

	err = sb.Terminate()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
}

func TestSandboxWithTunnels(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	app, err := modal.AppLookup(ctx, "libmodal-test", &modal.LookupOptions{
		CreateIfMissing: true,
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	image, err := app.ImageFromRegistry("alpine:3.21", nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Create a sandbox with port forwarding
	sandbox, err := app.CreateSandbox(image, &modal.SandboxOptions{
		Command:          []string{"cat"},
		EncryptedPorts:   []int{8443},
		UnencryptedPorts: []int{8080},
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(sandbox).ShouldNot(gomega.BeNil())
	g.Expect(sandbox.SandboxId).Should(gomega.HavePrefix("sb-"))

	defer sandbox.Terminate()

	tunnels, err := sandbox.Tunnels(30 * time.Second)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	g.Expect(tunnels).Should(gomega.HaveLen(2))

	// Test encrypted tunnel (port 8443)
	encryptedTunnel := tunnels[8443]
	g.Expect(encryptedTunnel.Host).Should(gomega.MatchRegexp(`\.modal\.host$`))
	g.Expect(encryptedTunnel.Port).Should(gomega.Equal(443))
	g.Expect(encryptedTunnel.URL()).Should(gomega.HavePrefix("https://"))

	host, port := encryptedTunnel.TLSSocket()
	g.Expect(host).Should(gomega.Equal(encryptedTunnel.Host))
	g.Expect(port).Should(gomega.Equal(encryptedTunnel.Port))

	// Test unencrypted tunnel (port 8080)
	unencryptedTunnel := tunnels[8080]
	g.Expect(unencryptedTunnel.UnencryptedHost).Should(gomega.MatchRegexp(`\.modal\.host$`))
	g.Expect(unencryptedTunnel.UnencryptedPort).Should(gomega.BeNumerically(">", 0))

	tcpHost, tcpPort, err := unencryptedTunnel.TCPSocket()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(tcpHost).Should(gomega.Equal(unencryptedTunnel.UnencryptedHost))
	g.Expect(tcpPort).Should(gomega.Equal(unencryptedTunnel.UnencryptedPort))
}

func TestCreateSandboxWithSecrets(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	secret, err := modal.SecretFromName(context.Background(), "libmodal-test-secret", &modal.SecretFromNameOptions{RequiredKeys: []string{"c"}})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	app, err := modal.AppLookup(context.Background(), "libmodal-test", &modal.LookupOptions{CreateIfMissing: true})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	image, err := app.ImageFromRegistry("alpine:3.21", nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	sb, err := app.CreateSandbox(image, &modal.SandboxOptions{Secrets: []*modal.Secret{secret}, Command: []string{"printenv", "c"}})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	output, err := io.ReadAll(sb.Stdout)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(string(output)).To(gomega.Equal("hello world\n"))
}
func TestSandboxPollAndReturnCode(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	app, err := modal.AppLookup(context.Background(), "libmodal-test", &modal.LookupOptions{CreateIfMissing: true})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	image, err := app.ImageFromRegistry("alpine:3.21", nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	sandbox, err := app.CreateSandbox(image, &modal.SandboxOptions{Command: []string{"cat"}})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	pollResult, err := sandbox.Poll()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(pollResult).Should(gomega.BeNil())

	// Send input to make the cat command complete
	_, err = sandbox.Stdin.Write([]byte("hello, sandbox"))
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	err = sandbox.Stdin.Close()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	waitResult, err := sandbox.Wait()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(waitResult).To(gomega.Equal(0))

	pollResult, err = sandbox.Poll()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(pollResult).ShouldNot(gomega.BeNil())
	g.Expect(*pollResult).To(gomega.Equal(0))
}

func TestSandboxPollAfterFailure(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	app, err := modal.AppLookup(context.Background(), "libmodal-test", &modal.LookupOptions{CreateIfMissing: true})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	image, err := app.ImageFromRegistry("alpine:3.21", nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	sandbox, err := app.CreateSandbox(image, &modal.SandboxOptions{
		Command: []string{"sh", "-c", "exit 42"},
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	waitResult, err := sandbox.Wait()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(waitResult).To(gomega.Equal(42))

	pollResult, err := sandbox.Poll()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(pollResult).ShouldNot(gomega.BeNil())
	g.Expect(*pollResult).To(gomega.Equal(42))
}

func TestCreateSandboxWithNetworkAccessParams(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	app, err := modal.AppLookup(ctx, "libmodal-test", &modal.LookupOptions{
		CreateIfMissing: true,
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	image, err := app.ImageFromRegistry("alpine:3.21", nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	sb, err := app.CreateSandbox(image, &modal.SandboxOptions{
		Command:       []string{"echo", "hello, network access"},
		BlockNetwork:  false,
		CIDRAllowlist: []string{"10.0.0.0/8", "192.168.0.0/16"},
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(sb).ShouldNot(gomega.BeNil())
	g.Expect(sb.SandboxId).Should(gomega.HavePrefix("sb-"))

	defer sb.Terminate()

	exitCode, err := sb.Wait()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(exitCode).Should(gomega.Equal(0))

	_, err = app.CreateSandbox(image, &modal.SandboxOptions{
		BlockNetwork:  false,
		CIDRAllowlist: []string{"not-an-ip/8"},
	})
	g.Expect(err).Should(gomega.HaveOccurred())
	g.Expect(err.Error()).Should(gomega.ContainSubstring("Invalid CIDR: not-an-ip/8"))

	_, err = app.CreateSandbox(image, &modal.SandboxOptions{
		BlockNetwork:  true,
		CIDRAllowlist: []string{"10.0.0.0/8"},
	})
	g.Expect(err).Should(gomega.HaveOccurred())
	g.Expect(err.Error()).Should(gomega.ContainSubstring("CIDRAllowlist cannot be used when BlockNetwork is enabled"))
}

func TestSandboxExecSecret(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	app, err := modal.AppLookup(context.Background(), "libmodal-test", &modal.LookupOptions{CreateIfMissing: true})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	image, err := app.ImageFromRegistry("alpine:3.21", nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	sb, err := app.CreateSandbox(image, nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(sb.SandboxId).ShouldNot(gomega.BeEmpty())
	defer sb.Terminate()

	secret, err := modal.SecretFromName(context.Background(), "libmodal-test-secret", &modal.SecretFromNameOptions{RequiredKeys: []string{"c"}})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	p, err := sb.Exec([]string{"printenv", "c"}, modal.ExecOptions{Stdout: modal.Pipe, Secrets: []*modal.Secret{secret}})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	buf, err := io.ReadAll(p.Stdout)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(string(buf)).Should(gomega.Equal("hello world\n"))
}

func TestSandboxFromId(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	app, err := modal.AppLookup(ctx, "libmodal-test", &modal.LookupOptions{CreateIfMissing: true})

	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	image, err := app.ImageFromRegistry("alpine:3.21", nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	sb, err := app.CreateSandbox(image, nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(sb.SandboxId).ShouldNot(gomega.BeEmpty())
	defer sb.Terminate()

	sbFromId, err := modal.SandboxFromId(ctx, sb.SandboxId)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(sbFromId.SandboxId).Should(gomega.Equal(sb.SandboxId))
}

func TestSandboxWithWorkdir(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	app, err := modal.AppLookup(ctx, "libmodal-test", &modal.LookupOptions{CreateIfMissing: true})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	image, err := app.ImageFromRegistry("alpine:3.21", nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	sb, err := app.CreateSandbox(image, &modal.SandboxOptions{
		Command: []string{"pwd"},
		Workdir: "/tmp",
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer sb.Terminate()

	output, err := io.ReadAll(sb.Stdout)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(string(output)).To(gomega.Equal("/tmp\n"))

	exitCode, err := sb.Wait()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(exitCode).To(gomega.Equal(0))

	_, err = app.CreateSandbox(image, &modal.SandboxOptions{
		Workdir: "relative/path",
	})
	g.Expect(err).Should(gomega.HaveOccurred())
	g.Expect(err.Error()).To(gomega.ContainSubstring("the Workdir value must be an absolute path"))
}

func TestSandboxSetTagsAndList(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	app, err := modal.AppLookup(ctx, "libmodal-test", &modal.LookupOptions{CreateIfMissing: true})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	image, err := app.ImageFromRegistry("alpine:3.21", nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	sb, err := app.CreateSandbox(image, nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer sb.Terminate()

	unique := fmt.Sprintf("%d", rand.Int())

	var before []string
	it, err := modal.SandboxList(ctx, &modal.SandboxListOptions{Tags: map[string]string{"test-key": unique}})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	for s, err := range it {
		g.Expect(err).ShouldNot(gomega.HaveOccurred())
		before = append(before, s.SandboxId)
	}
	g.Expect(before).To(gomega.HaveLen(0))

	err = sb.SetTags(map[string]string{"test-key": unique})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	var after []string
	it, err = modal.SandboxList(ctx, &modal.SandboxListOptions{Tags: map[string]string{"test-key": unique}})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	for s, err := range it {
		g.Expect(err).ShouldNot(gomega.HaveOccurred())
		after = append(after, s.SandboxId)
	}
	g.Expect(after).To(gomega.Equal([]string{sb.SandboxId}))
}

func TestSandboxSetMultipleTagsAndList(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	app, err := modal.AppLookup(ctx, "libmodal-test", &modal.LookupOptions{CreateIfMissing: true})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	image, err := app.ImageFromRegistry("alpine:3.21", nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	sb, err := app.CreateSandbox(image, nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer sb.Terminate()

	tagA := fmt.Sprintf("%d", rand.Int())
	tagB := fmt.Sprintf("%d", rand.Int())
	tagC := fmt.Sprintf("%d", rand.Int())

	err = sb.SetTags(map[string]string{"key-a": tagA, "key-b": tagB, "key-c": tagC})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	var ids []string
	it, err := modal.SandboxList(ctx, &modal.SandboxListOptions{Tags: map[string]string{"key-a": tagA}})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	for s, err := range it {
		g.Expect(err).ShouldNot(gomega.HaveOccurred())
		ids = append(ids, s.SandboxId)
	}
	g.Expect(ids).To(gomega.Equal([]string{sb.SandboxId}))

	ids = nil
	it, err = modal.SandboxList(ctx, &modal.SandboxListOptions{Tags: map[string]string{"key-a": tagA, "key-b": tagB}})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	for s, err := range it {
		g.Expect(err).ShouldNot(gomega.HaveOccurred())
		ids = append(ids, s.SandboxId)
	}
	g.Expect(ids).To(gomega.Equal([]string{sb.SandboxId}))

	ids = nil
	it, err = modal.SandboxList(ctx, &modal.SandboxListOptions{Tags: map[string]string{"key-a": tagA, "key-b": tagB, "key-d": "not-set"}})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	for s, err := range it {
		g.Expect(err).ShouldNot(gomega.HaveOccurred())
		ids = append(ids, s.SandboxId)
	}
	g.Expect(ids).To(gomega.HaveLen(0))
}

func TestSandboxListByAppId(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()

	app, err := modal.AppLookup(ctx, "libmodal-test", &modal.LookupOptions{CreateIfMissing: true})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	image, err := app.ImageFromRegistry("alpine:3.21", nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	sb, err := app.CreateSandbox(image, nil)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer sb.Terminate()

	count := 0
	it, err := modal.SandboxList(ctx, &modal.SandboxListOptions{AppId: app.AppId})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	for s, err := range it {
		g.Expect(err).ShouldNot(gomega.HaveOccurred())
		g.Expect(s.SandboxId).Should(gomega.HavePrefix("sb-"))
		count++
		if count >= 1 {
			break
		}
	}
	g.Expect(count).ToNot(gomega.Equal(0))
}
