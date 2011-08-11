#!/bin/bash

if [ "$1" == "start" ]; then 
        cd cmd/flunkymasterd/ && exec flunkymaster -F $2 &
        cd cmd/powerd/ && exec powerd -F $2 &
        cd cmd/heckled/ && exec heckled -F $2 &
elif [ "$1" == "stop" ]; then
        kill $(pidof flunkymaster)
	kill $(pidof powerd)
	kill $(pidof heckled)
else
       echo "NO arguments specified"
fi

