NAME := jot
GO ?= go
GOOS ?= linux
GOARCH ?= amd64
CGO_ENABLED := 0
DOCKER ?= docker
BINDIR ?= bin
BIN ?= $(BINDIR)/$(NAME)
CMD ?= ./cmd/$(NAME)
MKDIR_P ?= mkdir -p
# LDFLAGS ?= -X github.com/kyleterry/jot/version.Version=$(VERSION) -X github.com/kyleterry/jot/version.Commit=$(COMMIT) -w -s

.PHONY: all test build build-images clean
all: test build
test: ; GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) test -mod=vendor ./...
build: $(BIN)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) $(GO) build -ldflags $(LDFLAGS) -a -installsuffix cgo -mod=vendor -o $(BIN) $(CMD)
build-images: ; ./build-images
clean: ; rm -rf $(BINDIR)

$(BIN):
	$(MKDIR_P) $(BINDIR)
