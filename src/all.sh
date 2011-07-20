#!/bin/sh

mk () {
    pushd $1
    gomake $2
    popd
}

for target in pkg/daemon pkg/net pkg/interfaces cmd/flunky cmd/fctl cmd/diagd cmd/powerd; do
    mk $target $1
done
