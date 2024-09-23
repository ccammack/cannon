#!/bin/sh

go build -o ./cannon.exe  ./cmd/cannon/main.go
go build -o ./cannond.exe ./cmd/cannond/main.go

cp ./cannon.exe  ~/go/bin
cp ./cannond.exe ~/go/bin
