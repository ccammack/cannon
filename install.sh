#!/bin/sh

go build -o ./cannon.exe  ./cmd/client/main.go
go build -o ./cannond.exe ./cmd/server/main.go

cp ./cannon.exe  ~/go/bin
cp ./cannond.exe ~/go/bin
