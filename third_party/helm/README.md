# Third Party Helm Charts

These are currently all third party used helm charts.

## syslog

It's using an rsyslog build based on alpine Linux with log rotation.
The Dockerfile and helm chart are from https://github.com/lawesson/rsyslog-server.git

## ntp

It's using chrony from an image from Dockerhub from cturra.
The helm chart is from https://github.com/greg-redefined/chronyd-kubernetes.git
