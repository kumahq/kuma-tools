# kuma-tools
Various Kuma tools, scripts, YAMLs etc. that are not ready to put into official kuma repo

The goal of this repo is to provide a set of tools to experiment with Kuma.

## generate_mesh.go

This command line tool generates yaml configuration to create an arbitrary sized cluster.

- A DAG is generated which represents the network of services.
- Each dataplane will run an instance of [fake-service](https://github.com/nicholasjackson/fake-service).
- As an entry point it will run [slow_cooker](https://github.com/BuoyantIO/slow_cooker) to simulate load.

Here's an example that create 20 services:

```shell
go run generate_mesh.go -numServices 20 -percentEdge 33 -withGenerator -namespace kuma-test | kubectl apply -f -
```
