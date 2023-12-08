#!/bin/bash
set +x
BASEDIR=$(dirname "$0")
echo $BASEDIR
WORKING_DIR=$BASEDIR/protos

OUTPUT_DIR=.
mkdir -p $OUTPUT_DIR
for file in $(find $WORKING_DIR -type f -iname '*.proto'); do
	if [[ -f $file ]]; then
		echo "Running file" $file
		protoc -I=protos/ --go_out=$OUTPUT_DIR --go-grpc_out=$OUTPUT_DIR $file
	fi
done
