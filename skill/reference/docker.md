# coily docker - full reference

Mirrors `docker`. Underlying version at scan time: Docker version 28.0.1, build 068a01e

Command shape: `coily docker <verb...> [flags]`. Flags match the underlying CLI.

## `coily docker build`

### `coily docker build`

Start a build

Flags:

- `--add-host`
- `--allow`
- `--annotation`
- `--attest`
- `--build-arg`
- `--build-context`
- `--builder`
- `--cache-from`
- `--cache-to`
- `--call`
- `--cgroup-parent`
- `--check`
- `--debug`
- `--file`
- `--iidfile`
- `--label`
- `--load`
- `--metadata-file`
- `--network`
- `--no-cache`
- `--no-cache-filter`
- `--output`
- `--platform`
- `--progress`
- `--provenance`
- `--pull`
- `--push`
- `--quiet`
- `--sbom`
- `--secret`
- `--shm-size`
- `--ssh`
- `--tag`
- `--target`
- `--ulimit`

## `coily docker commit`

### `coily docker commit`

Usage:  docker commit [OPTIONS] CONTAINER [REPOSITORY[:TAG]]

Flags:

- `--author`
- `--change`
- `--message`
- `--pause`

## `coily docker container`

### `coily docker container` (group)

Usage:  docker container COMMAND

Subcommands: `attach`, `commit`, `cp`, `create`, `diff`, `exec`, `export`, `inspect`, `kill`, `logs`, `ls`, `pause`, `port`, `prune`, `rename`, `restart`, `rm`, `run`, `start`, `stats`, `stop`, `top`, `unpause`, `update`, `wait`

### `coily docker container attach`

Usage:  docker container attach [OPTIONS] CONTAINER

Flags:

- `--detach-keys`
- `--no-stdin`
- `--sig-proxy`

### `coily docker container commit`

Usage:  docker container commit [OPTIONS] CONTAINER [REPOSITORY[:TAG]]

Flags:

- `--author`
- `--change`
- `--message`
- `--pause`

### `coily docker container cp`

Usage:  docker container cp [OPTIONS] CONTAINER:SRC_PATH DEST_PATH|-

Flags:

- `--archive`
- `--follow-link`
- `--quiet`

### `coily docker container create`

Usage:  docker container create [OPTIONS] IMAGE [COMMAND] [ARG...]

Flags:

- `--add-host`
- `--annotation`
- `--attach`
- `--blkio-weight`
- `--blkio-weight-device`
- `--cap-add`
- `--cap-drop`
- `--cgroup-parent`
- `--cgroupns`
- `--cidfile`
- `--cpu-count`
- `--cpu-percent`
- `--cpu-period`
- `--cpu-quota`
- `--cpu-rt-period`
- `--cpu-rt-runtime`
- `--cpu-shares`
- `--cpus`
- `--cpuset-cpus`
- `--cpuset-mems`
- `--device`
- `--device-cgroup-rule`
- `--device-read-bps`
- `--device-read-iops`
- `--device-write-bps`
- `--device-write-iops`
- `--disable-content-trust`
- `--dns`
- `--dns-option`
- `--dns-search`
- `--domainname`
- `--entrypoint`
- `--env`
- `--env-file`
- `--expose`
- `--gpus`
- `--group-add`
- `--health-cmd`
- `--health-interval`
- `--health-retries`
- `--health-start-interval`
- `--health-start-period`
- `--health-timeout`
- `--help`
- `--hostname`
- `--init`
- `--interactive`
- `--io-maxbandwidth`
- `--io-maxiops`
- `--ip`
- `--ip6`
- `--ipc`
- `--isolation`
- `--kernel-memory`
- `--label`
- `--label-file`
- `--link`
- `--link-local-ip`
- `--log-driver`
- `--log-opt`
- `--mac-address`
- `--memory`
- `--memory-reservation`
- `--memory-swap`
- `--memory-swappiness`
- `--mount`
- `--name`
- `--network`
- `--network-alias`
- `--no-healthcheck`
- `--oom-kill-disable`
- `--oom-score-adj`
- `--pid`
- `--pids-limit`
- `--platform`
- `--privileged`
- `--publish`
- `--publish-all`
- `--pull`
- `--quiet`
- `--read-only`
- `--restart`
- `--rm`
- `--runtime`
- `--security-opt`
- `--shm-size`
- `--stop-signal`
- `--stop-timeout`
- `--storage-opt`
- `--sysctl`
- `--tmpfs`
- `--tty`
- `--ulimit`
- `--user`
- `--userns`
- `--uts`
- `--volume`
- `--volume-driver`
- `--volumes-from`
- `--workdir`

### `coily docker container diff`

Usage:  docker container diff CONTAINER

### `coily docker container exec`

Usage:  docker container exec [OPTIONS] CONTAINER COMMAND [ARG...]

Flags:

- `--detach`
- `--detach-keys`
- `--env`
- `--env-file`
- `--interactive`
- `--privileged`
- `--tty`
- `--user`
- `--workdir`

### `coily docker container export`

Usage:  docker container export [OPTIONS] CONTAINER

Flags:

- `--output`

### `coily docker container inspect`

Usage:  docker container inspect [OPTIONS] CONTAINER [CONTAINER...]

Flags:

- `--format`
- `--size`

### `coily docker container kill`

Usage:  docker container kill [OPTIONS] CONTAINER [CONTAINER...]

Flags:

- `--signal`

### `coily docker container logs`

Usage:  docker container logs [OPTIONS] CONTAINER

Flags:

- `--details`
- `--follow`
- `--since`
- `--tail`
- `--timestamps`
- `--until`

### `coily docker container ls`

Usage:  docker container ls [OPTIONS]

Flags:

- `--all`
- `--filter`
- `--format`
- `--last`
- `--latest`
- `--no-trunc`
- `--quiet`
- `--size`

### `coily docker container pause`

Usage:  docker container pause CONTAINER [CONTAINER...]

### `coily docker container port`

Usage:  docker container port CONTAINER [PRIVATE_PORT[/PROTO]]

### `coily docker container prune`

Usage:  docker container prune [OPTIONS]

Flags:

- `--filter`
- `--force`

### `coily docker container rename`

Usage:  docker container rename CONTAINER NEW_NAME

### `coily docker container restart`

Usage:  docker container restart [OPTIONS] CONTAINER [CONTAINER...]

Flags:

- `--signal`
- `--timeout`

### `coily docker container rm`

Usage:  docker container rm [OPTIONS] CONTAINER [CONTAINER...]

Flags:

- `--force`
- `--link`
- `--volumes`

### `coily docker container run`

Usage:  docker container run [OPTIONS] IMAGE [COMMAND] [ARG...]

Flags:

- `--add-host`
- `--annotation`
- `--attach`
- `--blkio-weight`
- `--blkio-weight-device`
- `--cap-add`
- `--cap-drop`
- `--cgroup-parent`
- `--cgroupns`
- `--cidfile`
- `--cpu-count`
- `--cpu-percent`
- `--cpu-period`
- `--cpu-quota`
- `--cpu-rt-period`
- `--cpu-rt-runtime`
- `--cpu-shares`
- `--cpus`
- `--cpuset-cpus`
- `--cpuset-mems`
- `--detach`
- `--detach-keys`
- `--device`
- `--device-cgroup-rule`
- `--device-read-bps`
- `--device-read-iops`
- `--device-write-bps`
- `--device-write-iops`
- `--disable-content-trust`
- `--dns`
- `--dns-option`
- `--dns-search`
- `--domainname`
- `--entrypoint`
- `--env`
- `--env-file`
- `--expose`
- `--gpus`
- `--group-add`
- `--health-cmd`
- `--health-interval`
- `--health-retries`
- `--health-start-interval`
- `--health-start-period`
- `--health-timeout`
- `--help`
- `--hostname`
- `--init`
- `--interactive`
- `--io-maxbandwidth`
- `--io-maxiops`
- `--ip`
- `--ip6`
- `--ipc`
- `--isolation`
- `--kernel-memory`
- `--label`
- `--label-file`
- `--link`
- `--link-local-ip`
- `--log-driver`
- `--log-opt`
- `--mac-address`
- `--memory`
- `--memory-reservation`
- `--memory-swap`
- `--memory-swappiness`
- `--mount`
- `--name`
- `--network`
- `--network-alias`
- `--no-healthcheck`
- `--oom-kill-disable`
- `--oom-score-adj`
- `--pid`
- `--pids-limit`
- `--platform`
- `--privileged`
- `--publish`
- `--publish-all`
- `--pull`
- `--quiet`
- `--read-only`
- `--restart`
- `--rm`
- `--runtime`
- `--security-opt`
- `--shm-size`
- `--sig-proxy`
- `--stop-signal`
- `--stop-timeout`
- `--storage-opt`
- `--sysctl`
- `--tmpfs`
- `--tty`
- `--ulimit`
- `--user`
- `--userns`
- `--uts`
- `--volume`
- `--volume-driver`
- `--volumes-from`
- `--workdir`

### `coily docker container start`

Usage:  docker container start [OPTIONS] CONTAINER [CONTAINER...]

Flags:

- `--attach`
- `--checkpoint`
- `--checkpoint-dir`
- `--detach-keys`
- `--interactive`

### `coily docker container stats`

Usage:  docker container stats [OPTIONS] [CONTAINER...]

Flags:

- `--all`
- `--format`
- `--no-stream`
- `--no-trunc`

### `coily docker container stop`

Usage:  docker container stop [OPTIONS] CONTAINER [CONTAINER...]

Flags:

- `--signal`
- `--timeout`

### `coily docker container top`

Usage:  docker container top CONTAINER [ps OPTIONS]

### `coily docker container unpause`

Usage:  docker container unpause CONTAINER [CONTAINER...]

### `coily docker container update`

Usage:  docker container update [OPTIONS] CONTAINER [CONTAINER...]

Flags:

- `--blkio-weight`
- `--cpu-period`
- `--cpu-quota`
- `--cpu-rt-period`
- `--cpu-rt-runtime`
- `--cpu-shares`
- `--cpus`
- `--cpuset-cpus`
- `--cpuset-mems`
- `--memory`
- `--memory-reservation`
- `--memory-swap`
- `--pids-limit`
- `--restart`

### `coily docker container wait`

Usage:  docker container wait CONTAINER [CONTAINER...]

## `coily docker cp`

### `coily docker cp`

Usage:  docker cp [OPTIONS] CONTAINER:SRC_PATH DEST_PATH|-

Flags:

- `--archive`
- `--follow-link`
- `--quiet`

## `coily docker exec`

### `coily docker exec`

Usage:  docker exec [OPTIONS] CONTAINER COMMAND [ARG...]

Flags:

- `--detach`
- `--detach-keys`
- `--env`
- `--env-file`
- `--interactive`
- `--privileged`
- `--tty`
- `--user`
- `--workdir`

## `coily docker image`

### `coily docker image` (group)

Usage:  docker image COMMAND

Subcommands: `build`, `history`, `import`, `inspect`, `load`, `ls`, `prune`, `pull`, `push`, `rm`, `save`, `tag`

### `coily docker image build`

Start a build

Flags:

- `--add-host`
- `--allow`
- `--annotation`
- `--attest`
- `--build-arg`
- `--build-context`
- `--builder`
- `--cache-from`
- `--cache-to`
- `--call`
- `--cgroup-parent`
- `--check`
- `--debug`
- `--file`
- `--iidfile`
- `--label`
- `--load`
- `--metadata-file`
- `--network`
- `--no-cache`
- `--no-cache-filter`
- `--output`
- `--platform`
- `--progress`
- `--provenance`
- `--pull`
- `--push`
- `--quiet`
- `--sbom`
- `--secret`
- `--shm-size`
- `--ssh`
- `--tag`
- `--target`
- `--ulimit`

### `coily docker image history`

Usage:  docker image history [OPTIONS] IMAGE

Flags:

- `--format`
- `--human`
- `--no-trunc`
- `--platform`
- `--quiet`

### `coily docker image import`

Usage:  docker image import [OPTIONS] file|URL|- [REPOSITORY[:TAG]]

Flags:

- `--change`
- `--message`
- `--platform`

### `coily docker image inspect`

Usage:  docker image inspect [OPTIONS] IMAGE [IMAGE...]

Flags:

- `--format`

### `coily docker image load`

Usage:  docker image load [OPTIONS]

Flags:

- `--input`
- `--platform`
- `--quiet`

### `coily docker image ls`

Usage:  docker image ls [OPTIONS] [REPOSITORY[:TAG]]

Flags:

- `--all`
- `--digests`
- `--filter`
- `--format`
- `--no-trunc`
- `--quiet`
- `--tree`

### `coily docker image prune`

Usage:  docker image prune [OPTIONS]

Flags:

- `--all`
- `--filter`
- `--force`

### `coily docker image pull`

Usage:  docker image pull [OPTIONS] NAME[:TAG|@DIGEST]

Flags:

- `--all-tags`
- `--disable-content-trust`
- `--platform`
- `--quiet`

### `coily docker image push`

Usage:  docker image push [OPTIONS] NAME[:TAG]

Flags:

- `--all-tags`
- `--disable-content-trust`
- `--platform`
- `--quiet`

### `coily docker image rm`

Usage:  docker image rm [OPTIONS] IMAGE [IMAGE...]

Flags:

- `--force`
- `--no-prune`

### `coily docker image save`

Usage:  docker image save [OPTIONS] IMAGE [IMAGE...]

Flags:

- `--output`
- `--platform`

### `coily docker image tag`

Usage:  docker image tag SOURCE_IMAGE[:TAG] TARGET_IMAGE[:TAG]

## `coily docker images`

### `coily docker images`

Usage:  docker images [OPTIONS] [REPOSITORY[:TAG]]

Flags:

- `--all`
- `--digests`
- `--filter`
- `--format`
- `--no-trunc`
- `--quiet`
- `--tree`

## `coily docker info`

### `coily docker info`

Usage:  docker info [OPTIONS]

Flags:

- `--format`

## `coily docker inspect`

### `coily docker inspect`

Usage:  docker inspect [OPTIONS] NAME|ID [NAME|ID...]

Flags:

- `--format`
- `--size`
- `--type`

## `coily docker kill`

### `coily docker kill`

Usage:  docker kill [OPTIONS] CONTAINER [CONTAINER...]

Flags:

- `--signal`

## `coily docker login`

### `coily docker login`

Usage:  docker login [OPTIONS] [SERVER]

Flags:

- `--password`
- `--password-stdin`
- `--username`

## `coily docker logout`

### `coily docker logout`

Usage:  docker logout [SERVER]

## `coily docker logs`

### `coily docker logs`

Usage:  docker logs [OPTIONS] CONTAINER

Flags:

- `--details`
- `--follow`
- `--since`
- `--tail`
- `--timestamps`
- `--until`

## `coily docker manifest`

### `coily docker manifest` (group)

Usage:  docker manifest COMMAND

Subcommands: `annotate`, `create`, `inspect`, `push`, `rm`

### `coily docker manifest annotate`

Usage:  docker manifest annotate [OPTIONS] MANIFEST_LIST MANIFEST

Flags:

- `--arch`
- `--os`
- `--os-features`
- `--os-version`
- `--variant`

### `coily docker manifest create`

Usage:  docker manifest create MANIFEST_LIST MANIFEST [MANIFEST...]

Flags:

- `--amend`
- `--insecure`

### `coily docker manifest inspect`

Usage:  docker manifest inspect [OPTIONS] [MANIFEST_LIST] MANIFEST

Flags:

- `--insecure`
- `--verbose`

### `coily docker manifest push`

Usage:  docker manifest push [OPTIONS] MANIFEST_LIST

Flags:

- `--insecure`
- `--purge`

### `coily docker manifest rm`

Usage:  docker manifest rm MANIFEST_LIST [MANIFEST_LIST...]

## `coily docker network`

### `coily docker network` (group)

Usage:  docker network COMMAND

Subcommands: `connect`, `create`, `disconnect`, `inspect`, `ls`, `prune`, `rm`

### `coily docker network connect`

Usage:  docker network connect [OPTIONS] NETWORK CONTAINER

Flags:

- `--alias`
- `--driver-opt`
- `--gw-priority`
- `--ip`
- `--ip6`
- `--link`
- `--link-local-ip`

### `coily docker network create`

Usage:  docker network create [OPTIONS] NETWORK

Flags:

- `--attachable`
- `--aux-address`
- `--config-from`
- `--config-only`
- `--driver`
- `--gateway`
- `--ingress`
- `--internal`
- `--ip-range`
- `--ipam-driver`
- `--ipam-opt`
- `--ipv4`
- `--ipv6`
- `--label`
- `--opt`
- `--scope`
- `--subnet`

### `coily docker network disconnect`

Usage:  docker network disconnect [OPTIONS] NETWORK CONTAINER

Flags:

- `--force`

### `coily docker network inspect`

Usage:  docker network inspect [OPTIONS] NETWORK [NETWORK...]

Flags:

- `--format`
- `--verbose`

### `coily docker network ls`

Usage:  docker network ls [OPTIONS]

Flags:

- `--filter`
- `--format`
- `--no-trunc`
- `--quiet`

### `coily docker network prune`

Usage:  docker network prune [OPTIONS]

Flags:

- `--filter`
- `--force`

### `coily docker network rm`

Usage:  docker network rm NETWORK [NETWORK...]

Flags:

- `--force`

## `coily docker ps`

### `coily docker ps`

Usage:  docker ps [OPTIONS]

Flags:

- `--all`
- `--filter`
- `--format`
- `--last`
- `--latest`
- `--no-trunc`
- `--quiet`
- `--size`

## `coily docker pull`

### `coily docker pull`

Usage:  docker pull [OPTIONS] NAME[:TAG|@DIGEST]

Flags:

- `--all-tags`
- `--disable-content-trust`
- `--platform`
- `--quiet`

## `coily docker push`

### `coily docker push`

Usage:  docker push [OPTIONS] NAME[:TAG]

Flags:

- `--all-tags`
- `--disable-content-trust`
- `--platform`
- `--quiet`

## `coily docker restart`

### `coily docker restart`

Usage:  docker restart [OPTIONS] CONTAINER [CONTAINER...]

Flags:

- `--signal`
- `--timeout`

## `coily docker rm`

### `coily docker rm`

Usage:  docker rm [OPTIONS] CONTAINER [CONTAINER...]

Flags:

- `--force`
- `--link`
- `--volumes`

## `coily docker rmi`

### `coily docker rmi`

Usage:  docker rmi [OPTIONS] IMAGE [IMAGE...]

Flags:

- `--force`
- `--no-prune`

## `coily docker run`

### `coily docker run`

Usage:  docker run [OPTIONS] IMAGE [COMMAND] [ARG...]

Flags:

- `--add-host`
- `--annotation`
- `--attach`
- `--blkio-weight`
- `--blkio-weight-device`
- `--cap-add`
- `--cap-drop`
- `--cgroup-parent`
- `--cgroupns`
- `--cidfile`
- `--cpu-count`
- `--cpu-percent`
- `--cpu-period`
- `--cpu-quota`
- `--cpu-rt-period`
- `--cpu-rt-runtime`
- `--cpu-shares`
- `--cpus`
- `--cpuset-cpus`
- `--cpuset-mems`
- `--detach`
- `--detach-keys`
- `--device`
- `--device-cgroup-rule`
- `--device-read-bps`
- `--device-read-iops`
- `--device-write-bps`
- `--device-write-iops`
- `--disable-content-trust`
- `--dns`
- `--dns-option`
- `--dns-search`
- `--domainname`
- `--entrypoint`
- `--env`
- `--env-file`
- `--expose`
- `--gpus`
- `--group-add`
- `--health-cmd`
- `--health-interval`
- `--health-retries`
- `--health-start-interval`
- `--health-start-period`
- `--health-timeout`
- `--help`
- `--hostname`
- `--init`
- `--interactive`
- `--io-maxbandwidth`
- `--io-maxiops`
- `--ip`
- `--ip6`
- `--ipc`
- `--isolation`
- `--kernel-memory`
- `--label`
- `--label-file`
- `--link`
- `--link-local-ip`
- `--log-driver`
- `--log-opt`
- `--mac-address`
- `--memory`
- `--memory-reservation`
- `--memory-swap`
- `--memory-swappiness`
- `--mount`
- `--name`
- `--network`
- `--network-alias`
- `--no-healthcheck`
- `--oom-kill-disable`
- `--oom-score-adj`
- `--pid`
- `--pids-limit`
- `--platform`
- `--privileged`
- `--publish`
- `--publish-all`
- `--pull`
- `--quiet`
- `--read-only`
- `--restart`
- `--rm`
- `--runtime`
- `--security-opt`
- `--shm-size`
- `--sig-proxy`
- `--stop-signal`
- `--stop-timeout`
- `--storage-opt`
- `--sysctl`
- `--tmpfs`
- `--tty`
- `--ulimit`
- `--user`
- `--userns`
- `--uts`
- `--volume`
- `--volume-driver`
- `--volumes-from`
- `--workdir`

## `coily docker search`

### `coily docker search`

Usage:  docker search [OPTIONS] TERM

Flags:

- `--filter`
- `--format`
- `--limit`
- `--no-trunc`

## `coily docker start`

### `coily docker start`

Usage:  docker start [OPTIONS] CONTAINER [CONTAINER...]

Flags:

- `--attach`
- `--checkpoint`
- `--checkpoint-dir`
- `--detach-keys`
- `--interactive`

## `coily docker stop`

### `coily docker stop`

Usage:  docker stop [OPTIONS] CONTAINER [CONTAINER...]

Flags:

- `--signal`
- `--timeout`

## `coily docker system`

### `coily docker system` (group)

Usage:  docker system COMMAND

Subcommands: `df`, `events`, `info`, `prune`

### `coily docker system df`

Usage:  docker system df [OPTIONS]

Flags:

- `--format`
- `--verbose`

### `coily docker system events`

Usage:  docker system events [OPTIONS]

Flags:

- `--filter`
- `--format`
- `--since`
- `--until`

### `coily docker system info`

Usage:  docker system info [OPTIONS]

Flags:

- `--format`

### `coily docker system prune`

Usage:  docker system prune [OPTIONS]

Flags:

- `--all`
- `--filter`
- `--force`
- `--volumes`

## `coily docker tag`

### `coily docker tag`

Usage:  docker tag SOURCE_IMAGE[:TAG] TARGET_IMAGE[:TAG]

## `coily docker volume`

### `coily docker volume` (group)

Usage:  docker volume COMMAND

Subcommands: `create`, `inspect`, `ls`, `prune`, `rm`, `update`

### `coily docker volume create`

Usage:  docker volume create [OPTIONS] [VOLUME]

Flags:

- `--availability`
- `--driver`
- `--group`
- `--label`
- `--limit-bytes`
- `--opt`
- `--required-bytes`
- `--scope`
- `--secret`
- `--sharing`
- `--topology-preferred`
- `--topology-required`
- `--type`

### `coily docker volume inspect`

Usage:  docker volume inspect [OPTIONS] VOLUME [VOLUME...]

Flags:

- `--format`

### `coily docker volume ls`

Usage:  docker volume ls [OPTIONS]

Flags:

- `--cluster`
- `--filter`
- `--format`
- `--quiet`

### `coily docker volume prune`

Usage:  docker volume prune [OPTIONS]

Flags:

- `--all`
- `--filter`
- `--force`

### `coily docker volume rm`

Usage:  docker volume rm [OPTIONS] VOLUME [VOLUME...]

Flags:

- `--force`

### `coily docker volume update`

Usage:  docker volume update [OPTIONS] [VOLUME]

Flags:

- `--availability`
