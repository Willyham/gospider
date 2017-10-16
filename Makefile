.PHONY: test test-race all devdeps deps cover generate

TESTFILES=$$(go list $$(glide novendor) | grep -v "mocks\|cmd")

all: test install

fmt: ## Format Go code
	@gofmt -s -w `go list -f {{.Dir}} ./... | grep -v "/vendor/"`

imports:
	find . -name '*.go' | grep -v vendor | xargs goimports -w

test: fmt
	go test ${TESTFILES}

test-race:
	go test -race ${TESTFILES}

cover:
	gocov test ${TESTFILES}

generate:
	go generate -v -x ${TESTFILES}

install:
	go install github.com/Willyham/gospider/cmd/...

devdeps:
	go install code.urchin.us/coin/vendor/github.com/axw/gocov/gocov
	go install code.urchin.us/coin/vendor/github.com/AlekSi/gocov-xml
	go install code.urchin.us/coin/vendor/github.com/vektra/mockery/cmd/mockery/
