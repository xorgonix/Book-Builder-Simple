#gofmt -w main.go internal/**/*.go migrations/*.go
#go test ./...
#go run . serve



gofmt -w internal/**/*.go migrations/*.go
go test ./...
go run . serve
