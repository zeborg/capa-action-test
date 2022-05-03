#!/bin/bash
git clone https://github.com/kubernetes-sigs/image-builder.git
sed 's/capa-ami-/test-capa-ami-/' ./image-builder/images/capi/packer/ami/packer.json
cd image-builder/images/capi
make deps-ami
make build-ami-ubuntu-2004
