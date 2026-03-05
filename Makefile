NAME := jot
GO ?= go
GOOS ?= linux
GOARCH ?= amd64
DOCKER ?= docker
BINDIR ?= bin
BIN ?= $(BINDIR)/$(NAME)
CMD ?= ./cmd/$(NAME)
MKDIR_P ?= mkdir -p
# LDFLAGS ?= -X github.com/kyleterry/jot/version.Version=$(VERSION) -X github.com/kyleterry/jot/version.Commit=$(COMMIT) -w -s
LDFLAGS ?= "-X 'github.com/kyleterry/jot/pkg/version.Version=1.0.0' -X 'github.com/kyleterry/jot/pkg/version.Commit=abc666' -w -s"

.PHONY: all test build build-images clean
all: test build
test: ; $(GO) test ./...
build: $(BIN)
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -ldflags $(LDFLAGS) -o $(BIN) $(CMD)
build-images: ; ./build-images
clean: ; rm -rf $(BINDIR)

$(BIN):
	$(MKDIR_P) $(BINDIR)
