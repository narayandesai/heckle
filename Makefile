all: 
	test ! -d src/github.com/ziutek/kasia && mkdir -p src/github.com/ziutek && cd src/github.com/ziutek/ && git clone https://github.com/ziutek/kasia.go kasia && cd ../../..
	GOPATH=`pwd` go install flunky/...

clean:
	rm -fR bin/* pkg/* src/github.com

