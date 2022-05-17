# gcp-runbatch

gcp-runbatch is a tool for running a docker container in a transient GCE VM
instance. The instance will be deleted once the container exits. It can be
invoked as a command line tool, or as a GCP Cloud Function.

## Installation

### Mac

```
brew tap bitcomplete/tap
brew update
brew install gcp-runbatch
```

### Linux

Download the appropriate Linux archive from the [latest
release](https://github.com/bitcomplete/gcp-runbatch/releases/latest) and copy
the binary to your PATH.

## Usage

### Command line

To execute a Docker image in a dedicated VM, run the gcp-runbatch command as
follows:

```
gcp-runbatch \
  --project-id=PROJECT_ID \
  --zone=ZONE \
  --service-account=SERVICE_ACCOUNT \
  IMAGE
```

Once the image process exits, the VM will be deleted. `IMAGE` should be a
reference to a Docker image recognized by `docker run`, e.g. an image name on
hub.docker.com, or a full Artifact Registry image URL.

Here's an example invocation:

```
$ gcp-runbatch \
  --project-id=long-octane-350517 \
  --zone=us-central1-a \
  --service-account=1234567890-compute@developer.gserviceaccount.com \
  hello-world
Successfully started instance runbatch-38408320. To tail batch logs run:
CLOUDSDK_PYTHON_SITEPACKAGES=1 gcloud beta --project=long-octane-350517
logging tail 'logName="projects/long-octane-350517/logs/runbatch" AND
resource.labels.instance_id="runbatch-38408320"' --format='get(text_payload)'
```

Running the `gcloud beta logging` command that gets printed will allow you to
tail the command logs.

### GCP Cloud Function

It's straightforward to deploy runbatch as a Cloud Function. Clone the repo and
then deploy using `gcloud functions deploy`:

```
go mod vendor
gcloud functions deploy runbatch-start \
  --trigger-http \
  --no-allow-unauthenticated \
  --runtime=go116 \
  --entry-point=Function
```
