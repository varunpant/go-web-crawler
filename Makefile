
test:
	go test -v .

test-race:
	go test -race -v .

run:
	go run main.go .

run-race:
	go run -race main.go