#!/bin/bash

OS="$1"
shift
FLAGS="$@"

cd image-builder/images/capi
PACKER_FLAGS="$flags" make build-ami-$os
cd ../../..