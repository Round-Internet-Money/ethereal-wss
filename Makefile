examples: listen_to_everything

listen_to_everything:
	go build -o bin/listen_to_everything examples/listen_to_everything/main.go 

wss:
	go vet .
	go mod tidy
	go build