#!/bin/bash

# Terminate all dataplanes for given test id

usage() {
cat <<EOF
Terminate all dataplanes in a given test environment.
Usage: $0 [-d] [--debug] <test_id>
EOF
}

getDoomed() {
	if [[ ! -z $1 ]]
	then
		max_items="--max-items $1"
	fi
	aws ec2 describe-instances \
		--filters Name=tag:kuma-test-id,Values=${TEST_ID} \
		          Name=tag:kuma-role,Values=dp  \
		          Name=instance-state-code,Values=0,16 \
		--query 'Reservations[*].Instances[*].[InstanceId]' \
		--output text $max_items > "$tmpdir/doomed.txt"
}

options=$(getopt -l "debug,dryrun" -o "d" -- "$@")
if [[ ! $? -eq 0 ]]
then
	usage
	exit 1
fi

eval set -- "$options"
while true
do
case $1 in
--dryrun)
	dryrun=1
	;;
-d|--debug)
	debug=1
	;;
*)
	shift
	break
	;;
esac
shift
done

TEST_ID="${TEST_ID:-$1}"

if [[ -z $TEST_ID ]]
then
	echo "Missing test id"
	usage
	exit 1
fi

tmpdir=$(mktemp -d)

if [[ -z $tmpdir ]]
then
	echo "Error creating temp dir work area"
	exit 1
fi

getDoomed
echo -n Terminating `cat $tmpdir/doomed.txt | wc -l` instances...

if [[ -z $dryrun ]]
then
	batch=$(head -n 500 $tmpdir/doomed.txt)
	while [[ ! -z $batch ]]
	do
		echo $batch | xargs aws ec2 terminate-instances --instance-ids >/dev/null
		sleep 1
		echo -n .
		getDoomed 500
		batch=$(head -n 500 $tmpdir/doomed.txt)
	done
fi
echo

if [[ -z $debug ]]
then
	rm -rf $tmpdir
else
	echo "Debug mode preserving $tmpdir"
fi
