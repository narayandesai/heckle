all: flunky flunkymaster heckle power testHeckle

clean:
	rm -f *.6 6.out flunky flunkymaster heckle power testHeckle fctl

flunky: flunky.6 main.6
	6l -o $@ main.6

#fctl:   fctl.6 flunky.6
#	6l -o $@ fctl.6

flunkymaster: flunkymaster.6
	6l -o $@ flunkymaster.6

heckle: heckleTypes.6 heckleFuncs.6 heckle.6
	6l -o $@ heckle.6

power: power.6
	6l -o $@ power.6

testHeckle: testHeckle.6
	6l -o $@ testHeckle.6

%.6: %.go
	6g $*.go

