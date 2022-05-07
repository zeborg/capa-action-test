#!/bin/bash

OS="$1"
shift
FLAGS="$@"

cd image-builder/images/capi
PACKER_FLAGS="$FLAGS" make build-ami-$OS
cd ../../..