#!/bin/sh

mk () {
    pushd $1
    gomake $2
    popd
}

for pkg in pkg/daemon ; do
    mk $pkg $1
done