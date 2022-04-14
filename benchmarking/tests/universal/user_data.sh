#!/bin/bash
yes | apt update
yes | apt install awscli
yes | apt install jq

sudo -u ubuntu bash -c "
aws s3 cp s3://kuma-benchmarking/universal_init.sh /home/ubuntu/universal_init.sh
chmod a+x /home/ubuntu/universal_init.sh
/home/ubuntu/universal_init.sh
"
