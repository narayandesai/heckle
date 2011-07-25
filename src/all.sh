#!/bin/bash

mk () {
    pushd $1 >/dev/null && echo $1
    gomake $2
    popd >/dev/null
}

for target in pkg/daemon pkg/net pkg/interfaces cmd/flunky cmd/fctl cmd/diagd cmd/powerd cmd/heckled cmd/testHeckle cmd/flunkymasterd cmd/comstat; do
    mk $target $1
done
