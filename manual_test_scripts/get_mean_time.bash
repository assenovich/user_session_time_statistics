#!/bin/bash

if [ "$#" -ne 0 ]; then
    curl -X GET http://localhost:8080/meanTime?user_id=$1
else
    curl -X GET http://localhost:8080/meanTime
fi
