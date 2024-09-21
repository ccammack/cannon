go build -o ./cannon.exe  ./cmd/cannon/main.go
go build -o ./cannond.exe ./cmd/cannond/main.go

Copy-Item ./cannon.exe  $env:USERPROFILE/go/bin
Copy-Item ./cannond.exe $env:USERPROFILE/go/bin
