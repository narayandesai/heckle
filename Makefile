BINARIES	:= bin/comstat bin/diagd bin/fctl bin/flunky bin/halloc bin/heckled \
                   bin/hfree bin/hstat bin/pm bin/powerd bin/provisiond

all: 
	-test ! -d src/github.com/ziutek/kasia && mkdir -p src/github.com/ziutek && cd src/github.com/ziutek/ && git clone https://github.com/ziutek/kasia.go kasia && cd ../../..
	GOPATH=`pwd` go install flunky/...

clean:
	rm -fR ${BINARIES} pkg/* src/github.com

