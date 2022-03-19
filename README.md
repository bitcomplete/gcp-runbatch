# runbatch

runbatch is a tool for running a docker container in a transient GCE VM
instance. The instance will be deleted once the container exits. It can be
invoked in several ways described below.

## Go function `runbatch.Start()`

`Start` accepts an `Input` struct specifying at a minimum:

* GCP project ID where the VM instance will be created.
* GCP zone where the VM instance will be created.
* GCP service account that will run the workload.
* Fully qualified docker image name for the workload to run.

For more details see `runbatch.go`.

## Command line tool: `cmd/runbatch/main.go`

This is a wrapper around `Start` that accepts command line parameters for the
inputs described above. For more details run:
`go run cmd/runbatch/main.go --help`.

## GCP Cloud Function: `function.StartRunBatch`

This is a wrapper around `Start` that accepts its input as JSON. To deploy to
GCP, run:

```
go mod vendor
gcloud functions deploy runbatch-start \
  --trigger-http \
  --no-allow-unauthenticated \
  --runtime=go116 \
  --entry-point=Function
```
