clean:
	rm -f *.6 gobot 6.out

gobot: simpleclient.6 simpleclientmain.6
	6l -o gobot simpleclientmain.6

%.6: %.go
	6g $*.go