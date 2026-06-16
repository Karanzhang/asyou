module github.com/asyou/server

go 1.20

require (
	github.com/asyou/core v0.0.0
	github.com/golang-jwt/jwt/v5 v5.0.0
	golang.org/x/crypto v0.14.0
	github.com/mattn/go-sqlite3 v1.14.16
)

replace github.com/asyou/core => ../core
