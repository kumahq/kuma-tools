# Ansible Playbooks for installing Kuma in Universal mode

## Prerequisites

To start, you will need some hosts provisioned, and an Ansible
inventory file that lists them. One way to do manage this is to set the
[`ANSIBLE_CONFIG`](https://docs.ansible.com/ansible/latest/reference_appendices/config.html#envvar-ANSIBLE_CONFIG)
environment variable to both configure Ansible and point it to the right
inventory.

There is no need to install Ansible, though Python is assumed to
already be available. Running the Ansible tools in [`./bin`](./bin)
will automatically install Ansible into a virtualenv and execute the
expected tool.

## Roles

### Install Kuma ([Source](./roles/install-kuma))

* copies a Kuma build from the local system to the remote host
* installs Kuma binaries are installed in `/opt/kuma`
* adds the Kuma installation directory all local user logins PATH
* creates a `kuma` role account

Note that this role also installs Envoy.

### Control Plane ([Source](./roles/control-plane))

* Runs a `kuma-cp` system service
* Manages the `kuma-cp` configuration file at `/etc/kuma/kuma.conf`

This role runs a simple `kuma-cp` control plane with no persistent backing
store. This has two implications:

* any DP tokens are lost on restart
* all non-builtin resources are lost on restart

Losing all the DP tokens means having to restart all the DPs on every
`kuma-cp` restart, which is awkward. Since this is a development
environment, we just turn off DP authorization (this could also
be solved by [#2955](https://github.com/kumahq/kuma/issues/2955)).

In general, there should only be 1 host with the `control-plane` role.

### Data Plane ([Source](./roles/data-plane))

This role is a mixin that manages a `kuma-dp` instance to expose
a service.  The "data-plane" role should be included in the relevant
service role, and the following variables need to be set:

* _service_name_: name of the service to expose
* _service_port_: TCP port on which to expose the service
* _workload_port_: TCP port that the service is listening on

### Echo Server ([Source](./roles/echo-server))

This role manages a Kuma
[echo server](https://github.com/kumahq/kuma/tree/master/test/server)
and exposes it to the mesh as `kuma.io/service=echo-server`.
