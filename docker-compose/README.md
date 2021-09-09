# Docker-compose setup

The goal of this is to run a universal stack with docker-compose.

There's 2 steps:

- The kuma-stack in `core/` which will start a control-plane and initialize a docker network.
- Different possible examples of setups:
  - `not-transparent` start an app with a dp in non transparent mode
  - `transparent` start 2 apps that communicate together in transparent mode (you can query them on [localhost:9000](localhost:9000) from the host).

To run anything first:

```shell
pushd core
docker-compose up -d
popd
pushd transparent # or whatever folder you'd like
docker-compose up -d
popd
```
