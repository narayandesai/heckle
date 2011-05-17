all: flunky

clean:
	rm -f *.6 6.out flunky

flunky: flunky.6 main.6
	6l -o $@ main.6

%.6: %.go
	6g $*.go