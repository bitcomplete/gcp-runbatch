#!/usr/bin/python3

# This is the startup script for GCE VMs managed by runbatch.

import base64
import grp
import json
import logging
import os
import pwd
import select
import subprocess
import sys
import tempfile
import urllib.request

logging.basicConfig(level=logging.INFO)

# Set unbuffered so that we can see the output of docker run in real time.
os.environ["PYTHONUNBUFFERED"] = "1"

# GCERuntime is used to interact with GCE metadata service and Google Cloud
# APIs.
class GCERuntime(object):
    def __init__(self):
        # Fetch token before anything else since we need it to make API calls.
        url = "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token"
        token = self.call_google_api(url)
        self.access_token = token["access_token"]

        url = "http://metadata.google.internal/computeMetadata/v1/?recursive=true"
        metadata = self.call_google_api(url)
        instance = metadata["instance"]
        self.instance_name = instance["name"]
        self.zone = instance["zone"].split("/")[3]
        self.attributes = instance["attributes"]
        self.project_id = metadata["project"]["projectId"]

    def call_google_api(self, url, data=None, method="GET"):
        headers = {}
        if url.startswith("http://metadata.google.internal"):
            headers["Metadata-Flavor"] = "Google"
        elif "googleapis.com" in url:
            headers["Authorization"] = "Bearer " + self.access_token
        if data:
            headers["Content-Type"] = "application/json"
        req = urllib.request.Request(url, data, headers, method=method)
        with urllib.request.urlopen(req) as resp:
            body = resp.read()
            if resp.getcode() != 200:
                msg = "request failed with code: {} {} {}"
                raise RuntimeError(
                    msg.format(url, resp.getcode(), body.decode("utf-8").strip())
                )
        return json.loads(body)


# CloudLoggingHandler is a logging.Handler that outputs messages to Google Cloud
# Logging.
class CloudLoggingHandler(logging.Handler):  # Inherit from logging.Handler
    LEVEL_SEVERITY_MAP = {
        logging.DEBUG: "DEBUG",
        logging.INFO: "INFO",
        logging.WARNING: "WARNING",
        logging.ERROR: "ERROR",
        logging.CRITICAL: "CRITICAL",
    }

    def __init__(self, runtime):
        logging.Handler.__init__(self)
        self.runtime = runtime

    def emit(self, record):
        severity = CloudLoggingHandler.LEVEL_SEVERITY_MAP.get(
            record.levelname, "DEFAULT"
        )
        text_payload = self.format(record)
        self.runtime.call_google_api(
            "https://logging.googleapis.com/v2/entries:write",
            data=json.dumps(
                {
                    "logName": "projects/{}/logs/runbatch".format(
                        self.runtime.project_id
                    ),
                    "resource": {
                        "type": "gce_instance",
                        "labels": {
                            "instance_id": self.runtime.instance_name,
                            "project_id": self.runtime.project_id,
                            "zone": self.runtime.zone,
                        },
                    },
                    "entries": [
                        {
                            "severity": severity,
                            "textPayload": text_payload,
                        }
                    ],
                }
            ).encode("utf-8"),
            method="POST",
        )


# docker_run starts the docker container and streams its output to logging.
def docker_run(image, env_file):
    cmd = ("docker", "run", "-i", "--env-file", env_file, image)
    proc = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    rlist = [proc.stdout, proc.stderr]
    while rlist:
        readers = select.select(rlist, [], [])[0]
        for f in readers:
            line = f.readline()
            if not line:
                rlist.remove(f)
                continue
            line_str = line.decode("utf-8").rstrip()
            if f == proc.stderr:
                level = logging.ERROR
            else:
                level = logging.INFO
            logging.log(level, line_str)
    code = proc.wait()
    level = logging.INFO if code == 0 else logging.ERROR
    logging.log(level, "exited with code: %d", code)


# _write_json_env translates environment variables from the given JSON-encoded
# string into shell-style environment variable assignments. It writes these
# assignments to the given file-like object out.
def _write_json_env(out, json_env):
    env = json.loads(json_env)
    for k, v in env.items():
        # TODO: we eventually need to wrap value with single quote
        # to escape characters like $ or double quotes inside value
        out.write("{}={}\n".format(k, v))
    out.write("\n\n")


# _write_env_file reads the JSON-encoded environment variables configured for
# runtime and writes them to a temporary .env file. The caller is responsible
# for deleting the file.
def write_env_file(runtime):
    fd, env_file = tempfile.mkstemp(prefix="runbatch", suffix=".env")
    try:
        with os.fdopen(fd, "w") as f:
            secret_json_envs = runtime.attributes.get("runbatch-secret-json-envs")
            if secret_json_envs:
                for secret_name in secret_json_envs.split(","):
                    url = "https://secretmanager.googleapis.com/v1/{}:access"
                    secret_version = runtime.call_google_api(url.format(secret_name))
                    secret_json_env = base64.b64decode(
                        secret_version["payload"]["data"]
                    )
                    _write_json_env(f, secret_json_env)
            json_env = runtime.attributes.get("runbatch-json-env")
            if json_env:
                _write_json_env(f, json_env)
    finally:
        if sys.exc_info()[0]:
            os.remove(env_file)
    return env_file


def main():
    runtime = GCERuntime()
    logging.root.addHandler(CloudLoggingHandler(runtime))

    try:
        # Since this script runs as root, and root's home directory is
        # read-only in GCE we cannot setup the docker credentials in
        # ~/.docker/config.json. Instead we create a user called runbatch
        # who runs docker commands. Here we create that user and make them
        # part of the docker group.
        if os.system("useradd -mg docker runbatch"):
            raise RuntimeError("failed to create user runbatch")
        # Having created the dedicated user, we switch to that user.
        os.setgid(grp.getgrnam("docker").gr_gid)
        os.setuid(pwd.getpwnam("runbatch").pw_uid)

        cmd = "docker-credential-gcr configure-docker --registries us-central1-docker.pkg.dev"
        if os.system(cmd):
            raise RuntimeError("failed to configure docker credentials")

        env_file = write_env_file(runtime)
        try:
            docker_run(runtime.attributes["runbatch-image"], env_file)
        finally:
            os.remove(env_file)
    except:
        logging.exception("top level exception in main()")
    finally:
        url = "https://www.googleapis.com/compute/v1/projects/{project_id}/zones/{zone}/instances/{instance_name}"
        runtime.call_google_api(url.format(**runtime.__dict__), method="DELETE")


if __name__ == "__main__":
    main()
