BIN := fileviews

.PHONY: build run test vet clean

build:
	go build -o $(BIN) ./cmd/fileviews

run: build
	./$(BIN)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f $(BIN)
