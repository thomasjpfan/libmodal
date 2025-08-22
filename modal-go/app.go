package modal

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	pb "github.com/modal-labs/libmodal/modal-go/proto/modal_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// App references a deployed Modal App.
type App struct {
	AppId string
	Name  string
	ctx   context.Context
}

// LookupOptions are options for finding deployed Modal objects.
type LookupOptions struct {
	Environment     string
	CreateIfMissing bool
}

// DeleteOptions are options for deleting a named object.
type DeleteOptions struct {
	Environment string // Environment to delete the object from.
}

// EphemeralOptions are options for creating a temporary, nameless object.
type EphemeralOptions struct {
	Environment string // Environment to create the object in.
}

// SandboxOptions are options for creating a Modal Sandbox.
type SandboxOptions struct {
	CPU               float64                      // CPU request in physical cores.
	Memory            int                          // Memory request in MiB.
	GPU               string                       // GPU reservation for the sandbox (e.g. "A100", "T4:2", "A100-80GB:4").
	Timeout           time.Duration                // Maximum duration for the Sandbox.
	Workdir           string                       // Working directory of the sandbox.
	Command           []string                     // Command to run in the Sandbox on startup.
	Secrets           []*Secret                    // Secrets to inject into the Sandbox.
	Volumes           map[string]*Volume           // Mount points for Volumes.
	CloudBucketMounts map[string]*CloudBucketMount // Mount points for cloud buckets.
	EncryptedPorts    []int                        // List of encrypted ports to tunnel into the sandbox, with TLS encryption.
	H2Ports           []int                        // List of encrypted ports to tunnel into the sandbox, using HTTP/2.
	UnencryptedPorts  []int                        // List of ports to tunnel into the sandbox without encryption.
	BlockNetwork      bool                         // Whether to block all network access from the sandbox.
	CIDRAllowlist     []string                     // List of CIDRs the sandbox is allowed to access. Cannot be used with BlockNetwork.
	Cloud             string                       // Cloud provider to run the sandbox on.
	Regions           []string                     // Region(s) to run the sandbox on.
	Verbose           bool                         // Enable verbose logging.
	Proxy             *Proxy                       // Reference to a Modal Proxy to use in front of this Sandbox.
}

// ImageFromRegistryOptions are options for creating an Image from a registry.
type ImageFromRegistryOptions struct {
	Secret *Secret // Secret for private registry authentication.
}

// parseGPUConfig parses a GPU configuration string into a GPUConfig object.
// The GPU string format is "type" or "type:count" (e.g. "T4", "A100:2").
// Returns nil if gpu is empty, or an error if the format is invalid.
func parseGPUConfig(gpu string) (*pb.GPUConfig, error) {
	if gpu == "" {
		return nil, nil
	}

	gpuType := gpu
	count := uint32(1)

	if strings.Contains(gpu, ":") {
		parts := strings.SplitN(gpu, ":", 2)
		gpuType = parts[0]
		parsedCount, err := strconv.ParseUint(parts[1], 10, 32)
		if err != nil || parsedCount < 1 {
			return nil, fmt.Errorf("invalid GPU count: %s, value must be a positive integer", parts[1])
		}
		count = uint32(parsedCount)
	}

	return pb.GPUConfig_builder{
		Type:    0, // Deprecated field, but required by proto
		Count:   count,
		GpuType: strings.ToUpper(gpuType),
	}.Build(), nil
}

// AppLookup looks up an existing App, or creates an empty one.
func AppLookup(ctx context.Context, name string, options *LookupOptions) (*App, error) {
	if options == nil {
		options = &LookupOptions{}
	}
	var err error
	ctx, err = clientContext(ctx)
	if err != nil {
		return nil, err
	}

	creationType := pb.ObjectCreationType_OBJECT_CREATION_TYPE_UNSPECIFIED
	if options.CreateIfMissing {
		creationType = pb.ObjectCreationType_OBJECT_CREATION_TYPE_CREATE_IF_MISSING
	}

	resp, err := client.AppGetOrCreate(ctx, pb.AppGetOrCreateRequest_builder{
		AppName:            name,
		EnvironmentName:    environmentName(options.Environment),
		ObjectCreationType: creationType,
	}.Build())

	if status, ok := status.FromError(err); ok && status.Code() == codes.NotFound {
		return nil, NotFoundError{fmt.Sprintf("app '%s' not found", name)}
	}
	if err != nil {
		return nil, err
	}

	return &App{AppId: resp.GetAppId(), Name: name, ctx: ctx}, nil
}

// CreateSandbox creates a new Sandbox in the App with the specified image and options.
func (app *App) CreateSandbox(image *Image, options *SandboxOptions) (*Sandbox, error) {
	if options == nil {
		options = &SandboxOptions{}
	}

	image, err := image.build(app)
	if err != nil {
		return nil, err
	}

	gpuConfig, err := parseGPUConfig(options.GPU)
	if err != nil {
		return nil, err
	}

	if options.Workdir != "" && !strings.HasPrefix(options.Workdir, "/") {
		return nil, fmt.Errorf("the Workdir value must be an absolute path, got: %s", options.Workdir)
	}

	var volumeMounts []*pb.VolumeMount
	if options.Volumes != nil {
		volumeMounts = make([]*pb.VolumeMount, 0, len(options.Volumes))
		for mountPath, volume := range options.Volumes {
			volumeMounts = append(volumeMounts, pb.VolumeMount_builder{
				VolumeId:               volume.VolumeId,
				MountPath:              mountPath,
				AllowBackgroundCommits: true,
				ReadOnly:               volume.IsReadOnly(),
			}.Build())
		}
	}

	var cloudBucketMounts []*pb.CloudBucketMount
	if options.CloudBucketMounts != nil {
		cloudBucketMounts = make([]*pb.CloudBucketMount, 0, len(options.CloudBucketMounts))
		for mountPath, mount := range options.CloudBucketMounts {
			proto, err := mount.toProto(mountPath)
			if err != nil {
				return nil, err
			}
			cloudBucketMounts = append(cloudBucketMounts, proto)
		}
	}

	var openPorts []*pb.PortSpec
	for _, port := range options.EncryptedPorts {
		openPorts = append(openPorts, pb.PortSpec_builder{
			Port:        uint32(port),
			Unencrypted: false,
		}.Build())
	}
	for _, port := range options.H2Ports {
		openPorts = append(openPorts, pb.PortSpec_builder{
			Port:        uint32(port),
			Unencrypted: false,
			TunnelType:  pb.TunnelType_TUNNEL_TYPE_H2.Enum(),
		}.Build())
	}
	for _, port := range options.UnencryptedPorts {
		openPorts = append(openPorts, pb.PortSpec_builder{
			Port:        uint32(port),
			Unencrypted: true,
		}.Build())
	}

	var portSpecs *pb.PortSpecs
	if len(openPorts) > 0 {
		portSpecs = pb.PortSpecs_builder{
			Ports: openPorts,
		}.Build()
	}

	secretIds := []string{}
	for _, secret := range options.Secrets {
		if secret != nil {
			secretIds = append(secretIds, secret.SecretId)
		}
	}

	var networkAccess *pb.NetworkAccess
	if options.BlockNetwork {
		if len(options.CIDRAllowlist) > 0 {
			return nil, fmt.Errorf("CIDRAllowlist cannot be used when BlockNetwork is enabled")
		}
		networkAccess = pb.NetworkAccess_builder{
			NetworkAccessType: pb.NetworkAccess_BLOCKED,
			AllowedCidrs:      []string{},
		}.Build()
	} else if len(options.CIDRAllowlist) > 0 {
		networkAccess = pb.NetworkAccess_builder{
			NetworkAccessType: pb.NetworkAccess_ALLOWLIST,
			AllowedCidrs:      options.CIDRAllowlist,
		}.Build()
	} else {
		networkAccess = pb.NetworkAccess_builder{
			NetworkAccessType: pb.NetworkAccess_OPEN,
			AllowedCidrs:      []string{},
		}.Build()
	}

	schedulerPlacement := pb.SchedulerPlacement_builder{Regions: options.Regions}.Build()

	var proxyId *string
	if options.Proxy != nil {
		proxyId = &options.Proxy.ProxyId
	}

	var workdir *string
	if options.Workdir != "" {
		workdir = &options.Workdir
	}

	createResp, err := client.SandboxCreate(app.ctx, pb.SandboxCreateRequest_builder{
		AppId: app.AppId,
		Definition: pb.Sandbox_builder{
			EntrypointArgs: options.Command,
			ImageId:        image.ImageId,
			SecretIds:      secretIds,
			TimeoutSecs:    uint32(options.Timeout.Seconds()),
			Workdir:        workdir,
			NetworkAccess:  networkAccess,
			Resources: pb.Resources_builder{
				MilliCpu:  uint32(1000 * options.CPU),
				MemoryMb:  uint32(options.Memory),
				GpuConfig: gpuConfig,
			}.Build(),
			VolumeMounts:       volumeMounts,
			CloudBucketMounts:  cloudBucketMounts,
			OpenPorts:          portSpecs,
			CloudProviderStr:   options.Cloud,
			SchedulerPlacement: schedulerPlacement,
			Verbose:            options.Verbose,
			ProxyId:            proxyId,
		}.Build(),
	}.Build())

	if err != nil {
		return nil, err
	}

	return newSandbox(app.ctx, createResp.GetSandboxId()), nil
}

// ImageFromRegistry creates an Image from a registry tag.
//
// Deprecated: ImageFromRegistry is deprecated, use modal.NewImageFromRegistry instead
func (app *App) ImageFromRegistry(tag string, options *ImageFromRegistryOptions) (*Image, error) {
	return NewImageFromRegistry(tag, options).build(app)
}

// ImageFromAwsEcr creates an Image from an AWS ECR tag.
//
// Deprecated: ImageFromAwsEcr is deprecated, use modal.NewImageFromAwsEcr instead
func (app *App) ImageFromAwsEcr(tag string, secret *Secret) (*Image, error) {
	return NewImageFromAwsEcr(tag, secret).build(app)
}

// ImageFromGcpArtifactRegistry creates an Image from a GCP Artifact Registry tag.
//
// Deprecated: ImageFromGcpArtifactRegistry is deprecated, use modal.NewImageFromGcpArtifactRegistry instead
func (app *App) ImageFromGcpArtifactRegistry(tag string, secret *Secret) (*Image, error) {
	return NewImageFromGcpArtifactRegistry(tag, secret).build(app)
}
