all: flunky fctl

clean:
	rm -f *.6 6.out flunky

flunky: flunky.6 main.6
	6l -o $@ main.6

fctl:	fctl.6 flunky.6
	6l -o $@ fctl.6

%.6: %.go
	6g $*.go