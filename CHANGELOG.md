# Changelog

Both client libraries are pre-1.0, and they have separate versioning.

## Unreleased

- Add some changes here

## modal-js/v0.3.17, modal-go/v0.0.17

- Added support for more parameters to `Sandbox.create()`:
  - `blockNetwork`: Whether to block all network access from the sandbox.
  - `cidrAllowlist`: List of CIDRs the sandbox is allowed to access.
  - `gpu`: GPU reservation for the sandbox (e.g. "A100", "T4:2", "A100-80GB:4").
  - `cloud`: Cloud provider to run the sandbox on.
  - `regions`: Region(s) to run the sandbox on.
  - `verbose`: Enable verbose logging.
  - `proxy`: Connect a Modal Proxy to a Sandbox.
  - `workdir`: Set the working directory.
- Added support for mounting `CloudBucketMount`s to Sandboxes.
- Added top level for Image objects that are lazy. The images are built when creating a sandbox.
  - `Image.fromRegistry` in typescript and `NewImageFromRegistry` in golang.
  - `Image.fromAwsEcr` in typescript and `NewImageFromAwsEcr` in golang.
  - `Image.fromGcpArtifactRegistry` in typescript and `NewImageFromGcpArtifactRegistry` in golang.
- Added `Secret.fromObject()` (JS) / `SecretFromMap()` (Go) to create a Secret from key-value pairs (like `from_dict()` in Python).
- Added `name` field to `App`s, `Sandboxe`s, `Secret`s, `Volume`s, and `Queue`s.
- Added support for `Function.getCurrentStats()` (JS) / `Function.GetCurrentStats()` (Go).
- Added support for `Function.updateAutoscaler()` (JS) / `Function.UpdateAutoscaler()` (Go).
- Added support for `Function.getWebURL()` (JS) / `Function.GetWebURL()` (Go).
- Added support for `Volume.readOnly()` (JS) / `Volume.ReadOnly()` (Go).
- Added support for setting tags on Sandboxes, and for listing Sandboxes (by tag).

## modal-js/v0.3.16, modal-go/v0.0.16

- Added support for getting Sandboxes from an ID.

## modal-js/v0.3.15, modal-go/v0.0.15

- Added support for snapshotting the filesystem of a Sandbox.
- Added support for polling Sandboxes to check if they are still running, or get the exit code.
- Added support to execute commands in Sandboxes with Secrets.
- Added support for creating Sandboxes with secrets.

## modal-js/v0.3.14, modal-go/v0.0.14

- Added support for setting up Tunnels to expose live TCP ports for Sandboxes.

## modal-js/v0.3.13, modal-go/v0.0.13

- Fixed calls of Cls with experimental `input_plane_region` option.
- (Go) Removed `Function.InputPlaneURL` from being exposed as public API.

## modal-js/v0.3.12, modal-go/v0.0.12

- Added support for passing a Secret to `imageFromRegistry()` (TS) / `ImageFromRegistry()` (Go) to pull images from private registries.
- Added support for pulling images from Google Artifact Registry with `imageFromGcpArtifactRegistry()` (TS) / `ImageFromGcpArtifactRegistry()` (Go).
- Added experimental support for calling remote functions deployed with the `input_plane_region` option in Python.

## modal-js/v0.3.11, modal-go/v0.0.11

- Add `InitializeClient()` (Go) / `initializeClient()` (TS) to initialize the client at runtime with credentials.
- Client libraries no longer panic at startup if no token ID / secret is provided. Instead, they will throw an error when trying to use the client.

## modal-js/v0.3.10, modal-go/v0.0.10

- Added `workdir` and `timeout` options to `ExecOptions` for Sandbox processes.

## modal-js/v0.3.9, modal-go/v0.0.9

- Added support for Sandbox filesystem.

## modal-js/v0.3.8

- Added support for CommonJS format / `require()`. Previously, modal-js only supported ESM `import`.

## modal-js/v0.3.7, modal-go/v0.0.8

- Added support for creating images from AWS ECR with `App.imageFromAwsEcr()` (TS) / `App.ImageFromAwsEcr()` (Go).
- Added support for accessing Modal secrets with `Secret.fromName()` (TS) / `modal.SecretFromName()` (Go).
- Fixed serialization of some pickled objects (negative ints, dicts) in modal-js.

## modal-js/v0.3.6, modal-go/v0.0.7

- Added support for the `Queue` object to manage distributed FIFO queues.
  - Queues have a similar interface as Python, with `put()` and `get()` being the primary methods.
  - You can put structured objects onto Queues, with limited support for the pickle format.
- Added `InvalidError`, `QueueEmptyError`, and `QueueFullError` to support Queues.
- Fixed a bug in `modal-js` that produced incorrect bytecode for bytes objects.
- Options in the Go SDK now take pointer types, and can be `nil` for default values.

## modal-js/v0.3.5, modal-go/v0.0.6

- Added support for spawning functions with `Function_.spawn()` (TS) / `Function.Spawn()` (Go).

## modal-js/v0.3.4, modal-go/v0.0.5

- Added feature for looking up and calling remote classes via the `Cls` object.
- (Go) Removed the initial `ctx context.Context` argument from `Function.Remote()`.

## modal-js/v0.3.3, modal-go/v0.0.4

- Support calling remote functions with arguments greater than 2 MiB in byte payload size.

## modal-js/v0.3.2, modal-go/v0.0.3

- First public release
- Basic `Function`, `Sandbox`, `Image`, and `ContainerProcess` support
