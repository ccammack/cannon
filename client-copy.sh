#!/bin/sh

server="$HOME/work/public/github.com/ccammack/cannon"

if [ "$(pwd)" = "$server" ]; then
    echo "cannot run from source directory: $server"
    exit
fi

if [ "$(pwd)" != "$server.client" ]; then
    echo "cannnot run from anywhere but $server.client directory"
    exit
fi

rsync -av --exclude='.git' --exclude='cache' "$server/" .
