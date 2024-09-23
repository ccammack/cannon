#!/bin/sh

go build -o ./cannon  ./cmd/cannon/main.go
go build -o ./cannond ./cmd/cannond/main.go

cp ./cannon  ~/go/bin
cp ./cannond ~/go/bin
