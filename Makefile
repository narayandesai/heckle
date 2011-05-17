all: flunky

clean:
	rm -f *.6 6.out flunky

flunky: flunky.6 simpleclientmain.6
	6l -o $@ simpleclientmain.6

%.6: %.go
	6g $*.go