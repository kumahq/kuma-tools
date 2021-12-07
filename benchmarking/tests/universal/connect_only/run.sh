#!/bin/bash -e

# Assumptions - CP running, token file generated, all dataplanes can share token. Bootstrap file in local dir.

. ../env

usage() {
cat <<EOF
Launch N dataplanes and connect a running CP.
Usage: $0 <test_id> <CP IP> <token file path> <number of dataplanes>
EOF
}

launch() {
	num=$1
	test_id=$2
	aws ec2 run-instances --image-id ${DP_AMI_IMAGE} --count ${num} --instance-type ${DP_INSTANCE_TYPE} --security-group-ids "${DP_SECURITY_GROUP_IDS}"  --subnet-id $DP_SUBNET_ID --iam-instance-profile ${DP_IAM_INSTANCE_PROFILE} --key-name "${DP_KEY_NAME}" --associate-public-ip-address --user-data file://${DP_USER_DATA_FILE}  --tag-specification 'ResourceType=instance,Tags=[{Key=kuma-role,Value=dp},{Key=kuma-test-id,Value='${test_id}'}]'
}

if [[ ! $ENV_SOURCED -eq 1 ]]
then
	# XXX add README.md
	echo "Incomplete environment; please see README.md" >/dev/stderr
	exit 1
fi

if [[ "$#" -ne 4 ]]
then
	usage
	exit 1
fi

test_id="$1"
cp_ip="$2"
token_path="$3"
ndp="$4"
bootstrap_path="./bootstrap.sh"

if [[ ! $ndp =~ ^[0-9]+$ ]]
then
	echo "ERROR: Invalid number of dataplanes" > /dev/stderr
	usage >/dev/stderr
	exit 1
fi

if [[ ! -f "$token_path" ]]
then
	echo "ERROR: Token file not found" > /dev/stderr
	usage >/dev/stderr
	exit 1
fi

if [[ ! -f "$bootstrap_path" ]]
then
	echo "ERROR: Bootstrap file not found" > /dev/stderr
	usage >/dev/stderr
	exit 1
fi

bootstrap_url="s3://${DP_TEST_BUCKET}/tests/${test_id}/bootstrap.sh"
aws s3 cp "$bootstrap_path" "$bootstrap_url"
aws s3 cp "$token_path" "s3://${DP_TEST_BUCKET}/tests/${test_id}/token"
aws s3 cp "$DP_KUMA_PKG" "s3://${DP_TEST_BUCKET}/tests/${test_id}/kuma.tgz"

echo Filling work queue
tmpdir=$(mktemp -d)
echo $tmpdir

if [[ -z $tmpdir ]]
then
	echo "Error creating temp dir work area" > /dev/stderr
	exit 1
fi

for i in `seq 1 $ndp`
do
	msg_id=`uuid`
	echo '{"Id":"'$msg_id'", "MessageBody":"'$bootstrap_url\|${test_id} dp-${i} ${cp-ip}'", "MessageGroupId": "0"}' >> "$tmpdir/msg_list.txt"
done
jq --slurp . "$tmpdir/msg_list.txt" > "$tmpdir/messages.json"

aws sqs send-message-batch --queue-url ${DP_WORK_QUEUE_URL} --entries file://"${tmpdir}/messages.json"

rm -rf $tmpdir

# AWS allows 500 per request
echo Launching instances
for i in `seq $(($ndp/500))`
do
	echo Launching 500 dps.
	launch 500 $test_id
done

rest=$(($ndp%500))
if [[ ! $rest -eq 0 ]]
then
	echo Launching $rest dps.
	launch $rest $test_id
fi
