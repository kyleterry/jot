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

.PHONY: all test build build-images clean
all: test build
test: ; GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) test -mod=vendor ./...
build: $(BIN)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) $(GO) build -a -installsuffix cgo -mod=vendor -o $(BIN) $(CMD)
build-images: ; ./build-images
clean: ; rm -rf $(BINDIR)

$(BIN):
	$(MKDIR_P) $(BINDIR)
