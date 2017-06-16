#!/bin/bash

# Make sure we are in the correct directory, just in case
# this script is executed from somewhere else
cd $(dirname "${BASH_SOURCE[0]}")

docker run --rm \
    -v $(pwd):/go/src/github.com/loginoff/nutanix-backup \
    -w /go/src/github.com/loginoff/nutanix-backup \
    golang /bin/bash -c 'go get -v && go build -v'
