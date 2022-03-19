package runbatch

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

	// Needed for embedding startupScript.
	_ "embed"

	"google.golang.org/api/compute/v1"
)

//go:embed startup.py
var startupScript string

type Input struct {
	// ProjectID is the GCP project ID where the VM instance will be created.
	ProjectID string `json:"projectId"`

	// Zone is the GCP zone where the VM instance will be created.
	Zone string `json:"zone"`

	// ServiceAccount is the GCP service account that will run the workload.
	ServiceAccount string `json:"serviceAccount"`

	// MachinePrefix is a prefix for the VM instance name.
	MachinePrefix string `json:"machinePrefix"`

	// Image is the fully qualified docker image name for the workload to run.
	Image string `json:"image"`

	// JSONEnv is a JSON-encoded string of environment variables to set for the
	// container, e.g. `{"FOO":"foo","BAR":"bar"}`.
	JSONEnv string `json:"jsonEnv"`

	// SecretJSONEnvs is a comma separated list of references to Secret Manager
	// secrets, e.g.: `projects/12345/secrets/secret_name/versions/latest`. The
	// secret payloads are expected to be JSON-encoded strings of environment
	// variables to set for the container.
	SecretJSONEnvs []string `json:"secretJsonEnvs"`
}

type Output struct {
	InstanceName string `json:"instanceName"`
}

// Start runs the workload specified in input on a transient GCE VM instance.
func Start(ctx context.Context, input *Input) (*Output, error) {
	if input.ProjectID == "" {
		return nil, errors.New("project ID is required")
	}
	if input.Zone == "" {
		return nil, errors.New("zone is required")
	}
	if input.ServiceAccount == "" {
		return nil, errors.New("service account is required")
	}
	if input.Image == "" {
		return nil, errors.New("image is required")
	}
	if input.MachinePrefix == "" {
		input.MachinePrefix = "runbatch"
	}

	var buf [4]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		return nil, err
	}

	region := input.Zone[:len(input.Zone)-2]

	computeService, err := compute.NewService(ctx)
	if err != nil {
		return nil, err
	}

	secretJSONEnvs := strings.Join(input.SecretJSONEnvs, ",")
	metadataItems := []*compute.MetadataItems{
		{
			Key:   "startup-script",
			Value: &startupScript,
		},
		{
			Key:   "runbatch-image",
			Value: &input.Image,
		},
		{
			Key:   "runbatch-json-env",
			Value: &input.JSONEnv,
		},
		{
			Key:   "runbatch-secret-json-envs",
			Value: &secretJSONEnvs,
		},
	}

	instance := compute.Instance{
		Name:        fmt.Sprintf("%s-%x", input.MachinePrefix, buf),
		MachineType: fmt.Sprintf("projects/%s/zones/%s/machineTypes/e2-micro", input.ProjectID, input.Zone),
		Disks: []*compute.AttachedDisk{
			{
				AutoDelete: true,
				Boot:       true,
				Type:       "PERSISTENT",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					DiskSizeGb:  10,
					DiskType:    fmt.Sprintf("projects/%s/zones/%s/diskTypes/pd-balanced", input.ProjectID, input.Zone),
					SourceImage: "projects/cos-cloud/global/images/family/cos-stable",
				},
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				Subnetwork: fmt.Sprintf("projects/%s/regions/%s/subnetworks/default", input.ProjectID, region),
				AccessConfigs: []*compute.AccessConfig{
					{
						Name:        "External NAT",
						Type:        "ONE_TO_ONE_NAT",
						NetworkTier: "PREMIUM",
					},
				},
			},
		},
		Metadata: &compute.Metadata{
			Items: metadataItems,
		},
		ServiceAccounts: []*compute.ServiceAccount{
			{
				Email: input.ServiceAccount,
				Scopes: []string{
					"https://www.googleapis.com/auth/cloud-platform",
				},
			},
		},
	}

	_, err = computeService.Instances.Insert(input.ProjectID, input.Zone, &instance).Do()
	if err != nil {
		return nil, err
	}

	return &Output{
		InstanceName: instance.Name,
	}, nil
}
