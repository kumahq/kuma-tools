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

## Playbooks

| Playbook | Purpose |
| --- | --- | 
| [Install Binaries](roles/common/tasks/main.yml) | Copy binaries to all hosts from a local Kuma build. |

