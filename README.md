# cronetes

A simple daemon which creates jobs in Kubernetes at certain intervals.

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

The job must contain a valid kubernetes [Job](http://kubernetes.io/docs/api-reference/batch/v1/definitions/#_v1_job).

### Schedule format

Cronetes uses the [cron](github.com/robfig/cron) package for scheduling. See
[CRON Expression Format](https://godoc.org/github.com/robfig/cron#hdr-CRON_Expression_Format)
for valid `schedule` inputs.

### Docker

There is a cronetes docker images which you can use. It doesn't come with any
configuration, so you need to create a new Docker images which has the config
included:

```
FROM quay.io/wercker/cronetes
ADD config.yml /config.yml
ENV CRONETES_CONFIG=/config.yml
```

Then run `docker build .`, and use the resulting image.

It is also possible to pipe the config into docker when running the container:

```
cat config_example.yml | docker run -i quay.io/wercker/cronetes cron
```

### Kubernetes

The easiest way to run cronetes is as a kubernetes pod. This requires you
injecting the config into the docker container, either by creating a custom
docker images, or by injecting a volume.

TODO(bvdberg): add kubernetes example

## reap

cronetes only creates Jobs, it doesn't clean them. This binary comes with
another command called `reap`. This will query any Jobs that have the label
`cronetes=true` and will delete them.

TODO(bvdberg): add configuration example using cronetes to reap old jobs
