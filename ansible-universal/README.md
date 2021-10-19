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

## Putting It All Together

To begin, we need a set of machines that are accessible over SSH.

I'm going to use the
[pulimi-stacks](https://github.com/jpeach/pulumi-stacks)
repository to spin up the machines we need. The `aws-devel` stack creates
an isolated AWS VPC that can be accessed through a SSH bastion host and
generates a SSH configuration that Ansible can use.

```bash
$ git clone https://github.com/jpeach/pulumi-stacks.git
$ cd pulumi-stacks/aws-devel
```

To set the number of hosts in the dev environment, use `pulumi config`:

```bash
$ pulumi config set workload:instanceCount 4
$ pulumi config
KEY                     VALUE
aws:region              ap-southeast-2
workload:instanceCount  4
workload:instanceType   t2.2xlarge
```

Now run `pulumi up` to build the stack.

```bash
$ pulumi up
...
```

The stack has Pulumi outputs that publish the addresses of the machines
that were provisioned. At time of writing, this stack installs Fedora
34 as the operating system.

```bash
$ pulumi stack output
Current stack outputs (5):
    OUTPUT           VALUE
    bastion.addr     3.26.11.88
    workload.addr.0  172.16.2.6
    workload.addr.1  172.16.2.7
    workload.addr.2  172.16.2.8
    workload.addr.3  172.16.2.9
```

The Pulumi tooling also generates SSH keys and configuration to access
the workload machines by proxying through the bastion host. This is
generated in the `./ssh/` directory, and can be used with
[`ssh -F`](https://man.openbsd.org/ssh#F).

```bash
$ ssh -F ./ssh/config 172.16.2.6 uname -a
Warning: Permanently added '3.26.11.88' (ECDSA) to the list of known hosts.
Warning: Permanently added '172.16.2.6' (ECDSA) to the list of known hosts.
Linux ip-172-16-2-6.ap-southeast-2.compute.internal 5.11.12-300.fc34.x86_64 #1 SMP Wed Apr 7 16:31:13 UTC 2021 x86_64 x86_64 x86_64 GNU/Linux
```

The next piece of configuration we need is an Ansible inventory file that
assigns roles to each machine. In this example, we place the control plane
and gateway on one host each, and an echo server instance on every host.

```bash
$ cat inventory
[controlplane]
172.16.2.6

[echoserver]
172.16.2.6
172.16.2.7
172.16.2.8
172.16.2.9

[gateway]
172.16.2.9
```

Finally, we need a configuration file to tell Ansible how to use the
right SSH configuration, and where the inventory is. We use an absolute
path to the SSH configuration file so that SSH wil work from anywhere, but
we can use a relative path to the inventory because Ansible will resolve it
relative to the configuration file.

```bash
$ cat ansible.cfg
[defaults]
inventory = ./inventory
transport = ssh

[ssh_connection]
ssh_args = -F /home/jpeach/src/pulumi-stacks/aws-devel/ssh/config
```

We can use the
[`ANSIBLE_CONFIG`](https://docs.ansible.com/ansible/latest/reference_appendices/config.html#the-configuration-file)
environment variable to tell Ansible to always use this configuration file.

```bash
$ export ANSIBLE_CONFIG=$(pwd)/ansible.cfg
```

Now we are ready to switch to the directory where you have checked out
these files.

When you run the main playbook, ansible will try to copy Kuma binaries
from the build directory of your Kuma workspace. To tell Ansible where
to find the binaries, you need to set the `builddir` variable:

```bash
$ ./bin/ansible-playbook -e builddir=/path/to/kuma/build site.yml
...
172.16.2.6                 : ok=39   changed=10   unreachable=0    failed=0    skipped=1    rescued=0    ignored=0
172.16.2.7                 : ok=31   changed=10   unreachable=0    failed=0    skipped=1    rescued=0    ignored=0
172.16.2.8                 : ok=31   changed=10   unreachable=0    failed=0    skipped=1    rescued=0    ignored=0
172.16.2.9                 : ok=40   changed=14   unreachable=0    failed=0    skipped=1    rescued=0    ignored=0
```

At this point, you should have a working Kuma control plane in universal
mode, with one service and a gateway deployed. You can ssh into any
machine and use kumactl as the "kuma" user:

```bash
$ ssh -F ./ssh/config 172.16.2.9
Last login: Tue Oct 19 04:53:57 2021 from 172.16.1.44
[fedora@ip-172-16-2-9 ~]$ sudo su -l kuma
[kuma@ip-172-16-2-9 ~]$ kumactl get dataplanes
MESH      NAME                                                         TAGS                           AGE
default   echo-server-ip-172-16-2-6-ap-southeast-2-compute-internal    kuma.io/service=echo-server    6m
default   echo-server-ip-172-16-2-7-ap-southeast-2-compute-internal    kuma.io/service=echo-server    6m
default   echo-server-ip-172-16-2-8-ap-southeast-2-compute-internal    kuma.io/service=echo-server    6m
default   echo-server-ip-172-16-2-9-ap-southeast-2-compute-internal    kuma.io/service=echo-server    6m
default   edge-gateway-ip-172-16-2-9-ap-southeast-2-compute-internal   kuma.io/service=edge-gateway   1m
```
