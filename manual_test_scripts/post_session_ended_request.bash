#!/bin/bash

if [ "$#" -ne 2 ]; then
    echo "usage: post_session_ended_request.bash <user_id> <session_id>"
    exit
fi
timestamp=`date +%s%3N`
curl -X POST -d "{\"user_id\": \"$1\", \"session_id\": \"$2\", \"timestamp\": \"$timestamp\"}" http://localhost:8080/sessionEnded
