# cronetes

Cronetes is a simple daemon which creates jobs in Kubernetes at specified
intervals. It is designed as a provisional solution until Kubernetes
ScheduledJobs is released.  For this reason it is designed to be run as a
single instance. Running multiple instances (with the same configuration) will
result in the same jobs being executed.

## cron

By using the command `cron` you will launch the cron daemon. This daemon
requires a configuration file:

```
cronetes cron --config="config_example.yml"
```

It is also possible to pipe this configuration into the application using
stdin:

```
cat config_example.yml | cronetes cron
```

### Configuration

The configuration is a yaml file, which contains a array of jobs and their
cron schedule:

```
# Every minute
- schedule: 0 * * * * *
  job:
    apiVersion: batch/v1
    kind: Job
    metadata:
      name: sendemails
    spec:
      template:
        metadata:
          name: sendemails
        spec:
          restartPolicy: Never
          containers:
          - name: job
            image: debian
```

The job must contain a valid Kubernetes [Job](http://kubernetes.io/docs/api-reference/batch/v1/definitions/#_v1_job).

### Schedule format

Cronetes uses the [cron](github.com/robfig/cron) package for scheduling. See
[CRON Expression Format](https://godoc.org/github.com/robfig/cron#hdr-CRON_Expression_Format)
for valid `schedule` inputs.

## Docker image

There is a cronetes Docker image which you can use. It is hosted op quay.io:
`quay.io/wercker/cronetes`.  It doesn't come with any configuration, so you need
to create a new Docker image which has the config included.

### Wercker

You need to create a repository on GitHub or Bitbucket, and add this repository
to wercker. Next you need the following wercker.yml:

```yaml
box: debian

# build creates a config file. Currently it is a static file, but could be dynamic
build:
  steps:
    - termie/bash-template:
        name: render config files
        input: config.template.yml

    - script:
        name: forward config files
        code: mv config.yml $WERCKER_OUTPUT_DIR

# push-quay takes cronetes as the base container, adds the config and pushes it to quay
push-quay:
  box:
    id: quay.io/wercker/cronetes
    registry: https://quay.io
    # Overriding the entrypoint: see https://github.com/wercker/wercker/issues/218
    entrypoint: /bin/sh -c
    cmd: /bin/sh
  steps:
    - script:
        name: install apk packages
        code: |
          echo "@edge http://dl-cdn.alpinelinux.org/alpine/edge/main" >> /etc/apk/repositories
          apk update && apk add ca-certificates

    - script:
        name: forward config files
        code: mv config.yml /

    - internal/docker-push:
        repository: quay.io/wercker/custom-cronetes
        registry: https://quay.io
        username: $DOCKER_USERNAME
        password: $DOCKER_PASSWORD
        tag: $WERCKER_GIT_BRANCH-$WERCKER_GIT_COMMIT
        entrypoint: /cronetes
```

In the `build` pipeline you need to ensure that the configuration files are
available for the next `push` pipeline. This can be done by simply moving
static files, using a template, or some other dynamic way.

### docker build

It is also possible to create a new `docker build` to create a new image.

```
FROM quay.io/wercker/cronetes
ADD config.yml /config.yml
ENV CRONETES_CONFIG=/config.yml
```

Then run `docker build .`, and use the resulting image.

### docker run

Finally it possible to pipe the config into docker when running the container:

```
cat config_example.yml | docker run -i quay.io/wercker/cronetes cron
```

## Kubernetes

The easiest way to run cronetes is as a Kubernetes pod. This requires a Docker
image which has the configuration included (see [Docker image](#docker-image)).

```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: custom-cronetes
spec:
  replicas: 1
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 0
  template:
    metadata:
      labels:
        app: custom-cronetes
    spec:
      containers:
      - name: custom-cronetes
        image: quay.io/wercker/custom-cronetes:${WERCKER_GIT_BRANCH}-${WERCKER_GIT_COMMIT}
        args: [
          "--kube-in-cluster",
          "cron",
          "--config=/config.yml",
        ]
```

## reap

cronetes only creates Jobs, it doesn't clean them. This binary comes with
another command called `reap`. This will query any Jobs that have the label
`cronetes=true` and will delete them.

```yaml
- schedule: 0 0 * * * *
  job:
    apiVersion: batch/v1
    kind: Job
    metadata:
      name: reap-jobs
    spec:
      template:
        metadata:
          name: reap-jobs
        spec:
          restartPolicy: Never
          containers:
          - name: cronetes-reap
            image: quay.io/wercker/cronetes
            args: [
              "--kube-in-cluster",
              "reap"
            ]
```
