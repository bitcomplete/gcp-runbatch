// runbatch is a tool for running a docker container in a transient GCE VM
// instance. The instance will be deleted once the container exits.
//
// Usage: runbatch -project-id=PROJECT_ID -zone=ZONE -service-account=SERVICE_ACCOUNT -image=IMAGE

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	runbatch "github.com/bitcomplete/gcp-runbatch"
	"github.com/urfave/cli/v2"
)

func main() {
	var input runbatch.Input
	app := &cli.App{
		ArgsUsage:       "IMAGE",
		HideHelpCommand: true,
		Usage:           "a tool for running a docker container in a transient GCE VM instance",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "project-id",
				Usage:       "Project ID",
				Destination: &input.ProjectID,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "zone",
				Usage:       "Zone name",
				Destination: &input.Zone,
				Required:    true,
			},
			// TODO: Use the default compute service account if empty.
			&cli.StringFlag{
				Name:        "service-account",
				Usage:       "Service account email",
				Destination: &input.ServiceAccount,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "machine-prefix",
				Value:       "runbatch",
				Usage:       "Machine prefix",
				Destination: &input.MachinePrefix,
			},
			&cli.StringFlag{
				Name:        "json-env",
				Usage:       "Environment variables in JSON format",
				Destination: &input.JSONEnv,
			},
			&cli.StringSliceFlag{
				Name:  "secret-json-env",
				Usage: "Secret containing environment variables in JSON format",
			},
		},
		Action: func(c *cli.Context) error {
			input.SecretJSONEnvs = c.StringSlice("secret-json-env")
			if c.NArg() != 1 {
				return errors.New("missing required argument: IMAGE")
			}
			input.Image = c.Args().First()
			output, err := runbatch.Start(context.Background(), &input)
			if err != nil {
				return err
			}
			fmt.Printf("Successfully started instance %s. To tail batch logs run:\n", output.InstanceName)
			fmt.Printf(
				`CLOUDSDK_PYTHON_SITEPACKAGES=1 gcloud beta --project=%[1]s logging tail `+
					`'logName="projects/%[1]s/logs/runbatch" AND resource.labels.instance_id="%[2]s"'`+
					` --format='get(text_payload)'`,
				input.ProjectID,
				output.InstanceName,
			)
			fmt.Println()
			return nil
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
