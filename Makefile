all: 
	-test ! -d src/github.com/ziutek/kasia && mkdir -p src/github.com/ziutek && cd src/github.com/ziutek/ && git clone https://github.com/ziutek/kasia.go kasia && cd ../../..
	GOPATH=`pwd` go install flunky/...

clean:
	rm -fR bin/{comstat,diagd,fctl,flunky,halloc,heckled,hfree,hstat,pm,powerd,provisiond} pkg/* src/github.com

