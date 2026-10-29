BIN := fileviews

.PHONY: build run test vet clean testdir vulncheck

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

# vulncheck reports known vulnerabilities your code can reach, in modules
# and in the Go standard library of the toolchain used.
vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# testdir creates a folder with every kind of entry for the manual tests
# (ManualTests.md) and starts the app in it.
TESTDIR ?= /tmp/fileviews-test
testdir: build
	rm -rf $(TESTDIR)
	mkdir -p $(TESTDIR)/sub/deeper $(TESTDIR)/remove-me
	printf 'hello\n' > $(TESTDIR)/a.txt
	head -c 4300 /dev/zero > $(TESTDIR)/big.bin
	printf '#!/bin/sh\necho hi\n' > $(TESTDIR)/run.sh && chmod 755 $(TESTDIR)/run.sh
	touch $(TESTDIR)/.hidden
	ln -s a.txt $(TESTDIR)/link-to-file
	ln -s sub $(TESTDIR)/link-to-dir
	ln -s missing $(TESTDIR)/broken-link
	mkfifo $(TESTDIR)/pipe
	./$(BIN) $(TESTDIR)
