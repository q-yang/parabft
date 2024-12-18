#!/usr/bin/env bash

SERVER_PID_FILE=server.pid

SERVER_PID=$(cat "${SERVER_PID_FILE}");

if [ -z "${SERVER_PID}" ]; then
    echo "Process id for servers is written to location: {$SERVER_PID_FILE}"
    #go build ../server/
    ./server -sim=true -log_level=debug -algorithm=hotstuffzg &
    echo $! >> ${SERVER_PID_FILE}
else
    echo "Servers are already started in this folder."
    exit 0
fi
