
About: 
------

heckle is a minimal provisioning system. It provides the basic hooks
for orchestrating raw hardware provisioning. At a high level, heckle
provides tracking and control of node boot configurations through a
full system provisioning workflow, with support for event and error
reporting and node hang detection. 

heckle sits on top of the standard PXE boot process. The PXE process
loads iPXE, and configures it to talk to the heckle provisioning
server. This server can recognize clients based on their IP addresses,
and direct their activities through construction of iPXE configuration
directives. 

flunky is an agent that can be used to coordinate the node
provisioning process from inside the provisioning system booted on the
node. It is a client that can request information from the heckle
provisioning server, send informational and error messages upstream,
and execute server rendered scripts. 

We have built a basic imaging system using these components, built on
top of Tiny Core Linux. This system is also contained in this source
tree. 

To Build:
---------

To install go release.1 ([golang.org/doc/install/source](http://golang.org/doc/install/source)):
    
    hg clone -u release https://code.google.com/p/go
    cd go/src
    ./all.bash
    <add ../bin to $PATH>

Use the top level Makefile

    make all

Resulting binaries will be in the bin/ subdirectory.

To Run:
-------

This documentations needs to be written

Known Issues:
-------------

The node activity timer is currently not tracked properly.

License:
--------

Heckle is licensed under the simplified 2-clause BSD license. See
LICENSE for details.
