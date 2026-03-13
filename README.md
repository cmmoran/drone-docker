# drone-docker

[![Build Status](http://cloud.drone.io/api/badges/drone-plugins/drone-docker/status.svg)](http://cloud.drone.io/drone-plugins/drone-docker)
[![Gitter chat](https://badges.gitter.im/drone/drone.png)](https://gitter.im/drone/drone)
[![Join the discussion at https://discourse.drone.io](https://img.shields.io/badge/discourse-forum-orange.svg)](https://discourse.drone.io)
[![Drone questions at https://stackoverflow.com](https://img.shields.io/badge/drone-stackoverflow-orange.svg)](https://stackoverflow.com/questions/tagged/drone.io)
[![](https://images.microbadger.com/badges/image/plugins/docker.svg)](https://microbadger.com/images/plugins/docker "Get your own image badge on microbadger.com")
[![Go Doc](https://godoc.org/github.com/drone-plugins/drone-docker?status.svg)](http://godoc.org/github.com/drone-plugins/drone-docker)
[![Go Report](https://goreportcard.com/badge/github.com/drone-plugins/drone-docker)](https://goreportcard.com/report/github.com/drone-plugins/drone-docker)

Drone plugin uses Docker-in-Docker to build and publish Docker images to a container registry. For the usage information and a listing of the available options please take a look at [the docs](http://plugins.drone.io/drone-plugins/drone-docker/).

### Git Leaks

Run the following script to install git-leaks support to this repo.
```
chmod +x ./git-hooks/install.sh
./git-hooks/install.sh
```

## Build

Build the binaries with the following commands:

```console
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0
export GO111MODULE=on

go build -v -a -tags netgo -o release/linux/amd64/drone-docker ./cmd/drone-docker
go build -v -a -tags netgo -o release/linux/amd64/drone-gcr ./cmd/drone-gcr
go build -v -a -tags netgo -o release/linux/amd64/drone-ecr ./cmd/drone-ecr
go build -v -a -tags netgo -o release/linux/amd64/drone-acr ./cmd/drone-acr
go build -v -a -tags netgo -o release/linux/amd64/drone-heroku ./cmd/drone-heroku
go build -v -a -tags netgo -o release/linux/amd64/drone-gar ./cmd/drone-gar
```

## Docker

Build the Docker images with the following commands:

```console
docker build \
  --label org.label-schema.build-date=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  --label org.label-schema.vcs-ref=$(git rev-parse --short HEAD) \
  --file docker/docker/Dockerfile.linux.amd64 --tag plugins/docker .

docker build \
  --label org.label-schema.build-date=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  --label org.label-schema.vcs-ref=$(git rev-parse --short HEAD) \
  --file docker/gcr/Dockerfile.linux.amd64 --tag plugins/gcr .

docker build \
  --label org.label-schema.build-date=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  --label org.label-schema.vcs-ref=$(git rev-parse --short HEAD) \
  --file docker/ecr/Dockerfile.linux.amd64 --tag plugins/ecr .

docker build \
  --label org.label-schema.build-date=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  --label org.label-schema.vcs-ref=$(git rev-parse --short HEAD) \
  --file docker/acr/Dockerfile.linux.amd64 --tag plugins/acr .

docker build \
  --label org.label-schema.build-date=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  --label org.label-schema.vcs-ref=$(git rev-parse --short HEAD) \
  --file docker/heroku/Dockerfile.linux.amd64 --tag plugins/heroku .
  
docker build \
  --label org.label-schema.build-date=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  --label org.label-schema.vcs-ref=$(git rev-parse --short HEAD) \
  --file docker/gar/Dockerfile.linux.amd64 --tag plugins/gar .
```

## Usage

> Notice: Be aware that the Docker plugin currently requires privileged capabilities, otherwise the integrated Docker daemon is not able to start.

### Using Docker buildkit Secrets

```yaml
kind: pipeline
name: default

steps:
- name: build dummy docker file and publish
  image: plugins/docker
  pull: never
  settings:
    repo: tphoney/test
    tags: latest
    secret: id=mysecret,src=secret-file
    username:
      from_secret: docker_username
    password:
      from_secret: docker_password
```

### Publishing step outputs

This fork can publish selected plugin settings and runtime image results as
step outputs for downstream steps.

This feature is only useful with the corresponding `drone-runner-docker` fork
at `~/code/cmmoran/drone-runner-docker`. That runner injects the
`drone-output` helper, resolves downstream `from_output` references, and tells
the plugin which settings were provided via `from_secret` using
`PLUGIN_FROM_SECRET_KEYS`.

```yaml
steps:
- name: build-and-push
  image: plugins/docker
  settings:
    repo: octocat/hello-world
    tags:
    - latest
    - 1.2.3
    labels:
    - org.opencontainers.image.title=hello-world
    outputs:
    - tags
    - digest
    - image_refs
    - primary_image_ref
    - image_with_digest
    - outputs.image_repo=settings.repo

- name: use-image-metadata
  image: alpine
  environment:
    FIRST_TAG:
      from_output: build-and-push.tags.0
    DIGEST:
      from_output: build-and-push.digest
    IMAGE_REF:
      from_output: build-and-push.primary_image_ref
    IMAGE_WITH_DIGEST:
      from_output: build-and-push.image_with_digest
    IMAGE_REPO:
      from_output: build-and-push.outputs.image_repo
```

The `outputs` list supports these forms:

- `tags`
  Publishes a supported output source by name. For setting-backed sources, the
  plugin resolves this like `settings.tags`.
- `settings.repo`
  Publishes an explicit setting-backed source.
- `outputs.image_repo=settings.repo`
  Publishes the value of `settings.repo` as the `outputs.image_repo` output.

Downstream steps can then consume these values with `from_output`, for example
`build-and-push.tags.0`, `build-and-push.digest`, or
`build-and-push.outputs.image_repo`.

First-class runtime outputs:

- `digest`
  The pushed image digest, for example `sha256:...`
- `image_refs`
  The pushed tag refs as `repo:tag` values
- `primary_image_ref`
  The first pushed `repo:tag` value
- `image_with_digest`
  The canonical immutable ref as `repo@sha256:...`

Supported setting-backed outputs include:

- `repo`
- `tags`
- `labels`
- `label_schema`
- `dockerfile`
- `context`
- `args`
- `args_from_env`
- `target`
- `cache_from`
- `platform`
- `dry_run`
- `push_only`
- `source_image`
- `artifact_file`

Behavior:

- String settings are published as plain output values.
- List settings are published as indexed outputs.
  Example: `tags` becomes `tags.0`, `tags.1`, and so on.
- Boolean and numeric values are published as JSON-formatted values.
- Runtime outputs are only available when the step actually produced them.
  Example: `digest` requires a successful non-dry-run push.
- Invalid, blocked, or unavailable output sources fail the step.

Security guardrails:

- `settings.X` outputs are blocked when the runner marks `X` in
  `PLUGIN_FROM_SECRET_KEYS`
- when runner metadata is absent, the plugin falls back to a conservative
  sensitive-name denylist for registry credentials, access tokens, secret
  inputs, SSH keys, docker config, base image credentials, and cosign
  private-key/password fields

Using a dockerfile that references the secret-file 

```bash
# syntax=docker/dockerfile:1.2

FROM alpine

# shows secret from default secret location:
RUN --mount=type=secret,id=mysecret cat /run/secrets/mysecret
```

and a secret file called secret-file

```
COOL BANANAS
```


### Running from the CLI

```console
docker run --rm \
  -e PLUGIN_TAG=latest \
  -e PLUGIN_REPO=octocat/hello-world \
  -e DRONE_COMMIT_SHA=d8dbe4d94f15fe89232e0402c6e8a0ddf21af3ab \
  -v $(pwd):$(pwd) \
  -w $(pwd) \
  --privileged \
  plugins/docker --dry-run
```

### GAR (Google Artifact Registry)

```yaml
kind: pipeline
name: default
type: docker

steps:
  - name: push-to-gar
    image: plugins/gar
    pull: never
    settings:
      tag: latest
      repo: project-id/repo/image-name
      location: us
      json_key:
        from_secret: gcr_json_key
```

### GAR (Google Artifact Registry) using workload identity (OIDC)

```yaml
steps:
  - name: push-to-gar
    image: plugins/gar
    pull: never
    settings:
      tag: latest
      repo: project-id/repo/image-name
      location: europe
      project_number: project-number
      pool_id: workload identity pool id
      provider_id: workload identity provider id
      service_account_email: service account email
      oidc_token_id:
        from_secret: token 
```

## Developer Notes

- When updating the base image, you will need to update for each architecture and OS.
- Arm32 base images are no longer being updated.

## Release procedure

Run the changelog generator.

```BASH
docker run -it --rm -v "$(pwd)":/usr/local/src/your-app githubchangeloggenerator/github-changelog-generator -u drone-plugins -p drone-docker -t <secret github token>
```

You can generate a token by logging into your GitHub account and going to Settings -> Personal access tokens.

Next we tag the PR's with the fixes or enhancements labels. If the PR does not fufil the requirements, do not add a label.

Run the changelog generator again with the future version according to semver.

```BASH
docker run -it --rm -v "$(pwd)":/usr/local/src/your-app githubchangeloggenerator/github-changelog-generator -u drone-plugins -p drone-docker -t <secret token> --future-release v1.0.0
```

Create your pull request for the release. Get it merged then tag the release.
