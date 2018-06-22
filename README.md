# Tootgo
A rewrite of Tootbot (https://github.com/corbindavenport/tootbot) in Golang.

## Compiling
To compile tootgo, first run `go get -d ./...` to fetch dependencies, and then run `go build -ldflags="-s -w" -tags netgo` to compile the program into a static binary.
