.PHONY: all clean install

VERSION=$(shell git tag -l --points-at HEAD)
COMMIT=$(shell git rev-parse HEAD)
REPO="github.com/dollarshaveclub/psst"

all: clean bin/psst

install:
	go install -ldflags "-X $(REPO)/internal/version.CommitSHA=$(COMMIT) -X $(REPO)/internal/version.Version=$(VERSION)" .

build-dsc:
	GOOS=darwin GOARCH=`uname -m` go build -ldflags "-X $(REPO)/cmd.Version=$(VERSION) -X $(REPO)/cmd.CommitSHA=$(COMMIT) -X $(REPO)/cmd.CompiledDirectory=github -X $(REPO)/cmd.CompiledStorage=vault -X $(REPO)/cmd.Org=dollarshaveclub" -o bin/dsc/darwin/psst .

build-all:
	GOOS=darwin GOARCH=`uname -m` go build -ldflags "-X $(REPO)/cmd.Version=$(VERSION) -X $(REPO)/cmd.CommitSHA=$(COMMIT)" -o bin/psst-darwin .

release:
	cd releaser && go build && ./releaser --commit "$(COMMIT)" --release "${RELEASE}"

test:
	go test ./...

clean:
	rm -rf bin/*