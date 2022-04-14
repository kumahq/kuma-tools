#!/bin/bash -e

# Assumptions - CP running, token file generated, all dataplanes can share token. Bootstrap file in local dir.

. ../env.eks.benchmarking0

usage() {
cat <<EOF
Launch N EKS nodes and place in cluster.
Usage: $0 <test_id> <cluster endpoint> <CA data> <cluster name> <number of nodes>
EOF
}

launch() {
	num=$1
	test_id=$2
	cl_name=$3
	aws ec2 run-instances --image-id ${DP_AMI_IMAGE} --count ${num} --instance-type ${DP_INSTANCE_TYPE} --security-group-ids "${DP_SECURITY_GROUP_IDS}"  --subnet-id $DP_SUBNET_ID --iam-instance-profile ${DP_IAM_INSTANCE_PROFILE} --key-name "${DP_KEY_NAME}" --associate-public-ip-address --user-data file://${DP_USER_DATA_FILE}  --tag-specification 'ResourceType=instance,Tags=[{Key=kuma-role,Value=dp},{Key=kuma-test-id,Value='${test_id}'},{Key=kubernetes.io/cluster/'${cl_name}',Value=owned}]'
	#aws ec2 run-instances --image-id ${DP_AMI_IMAGE} --count ${num} --instance-type ${DP_INSTANCE_TYPE} --security-group-ids "${DP_SECURITY_GROUP_IDS}"  --subnet-id $DP_SUBNET_ID --iam-instance-profile ${DP_IAM_INSTANCE_PROFILE} --key-name "${DP_KEY_NAME}" --associate-public-ip-address --user-data file://${DP_USER_DATA_FILE}  --tag-specification 'ResourceType=instance,Tags=[{Key=kuma-role,Value=cp},{Key=kuma-test-id,Value='${test_id}'},{Key=kubernetes.io/cluster/'${cl_name}',Value=owned}]'
}

if [[ ! $ENV_SOURCED -eq 1 ]]
then
	# XXX add README.md
	echo "Incomplete environment; please see README.md" >/dev/stderr
	exit 1
fi

if [[ "$#" -ne 5 ]]
then
	usage
	exit 1
fi

test_id="$1"
cluster_endpoint="$2"
ca_data="$3"
cluster_name="$4"
ndp="$5"
bootstrap_path="./bootstrap.sh"

if [[ ! -f "$bootstrap_path" ]]
then
	echo "ERROR: Bootstrap file not found" > /dev/stderr
	usage >/dev/stderr
	exit 1
fi

bootstrap_url="s3://${DP_TEST_BUCKET}/tests/${test_id}/bootstrap.sh"
aws s3 cp "$bootstrap_path" "$bootstrap_url"

# AWS allows 500 per request
echo Launching instances
for i in `seq $(($ndp/500))`
do
	echo Launching 500 dps.
	launch 500 $test_id $cluster_name
done

rest=$(($ndp%500))
if [[ ! $rest -eq 0 ]]
then
	echo Launching $rest dps.
	launch $rest $test_id $cluster_name
fi
