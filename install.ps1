go build -o ./cannon.exe  ./cmd/client/main.go
go build -o ./cannond.exe ./cmd/server/main.go

Copy-Item ./cannon.exe  $env:USERPROFILE/go/bin
Copy-Item ./cannond.exe $env:USERPROFILE/go/bin
