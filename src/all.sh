#!/bin/sh

mk () {
    pushd $1
    gomake $2
    popd
}

for target in pkg/daemon pkg/net cmd/fctl cmd/diagd; do
    mk $target $1
done
