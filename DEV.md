# Developing gcp-runbatch

## Code entrypoints

### Go function `runbatch.Start()`

`Start` accepts an `Input` struct specifying at a minimum:

* GCP project ID where the VM instance will be created.
* GCP zone where the VM instance will be created.
* GCP service account that will run the workload.
* Fully qualified docker image name for the workload to run.

For more details see `runbatch.go`.

### Command line tool: `cmd/gcp-runbatch/main.go`

This is a wrapper around `Start` that accepts command line parameters for the
inputs described above. For more details run:
`go run cmd/gcp-runbatch/main.go --help`.

### GCP Cloud Function: `function.StartRunBatch`

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

## Release process

Make sure that the git repo is clean and up to date with origin/main. Then run:

```
(read -r v && git tag -a v$v -m v$v && git push origin v$v)
```
