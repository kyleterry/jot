NAME := jot
GO ?= go
GOOS ?= linux
CGO_ENABLED := 0
BINDIR ?= bin
BIN ?= $(BINDIR)/$(NAME)
CMD ?= ./cmd/$(NAME)

all: test build
test: ; $(GO) test ./...
build: ; GOOS=$(GOOS) CGO_ENABLED=$(CGO_ENABLED) $(GO) build -a -installsuffix cgo -o $(BIN) $(CMD)
clean: ; rm -rf $(BINDIR)

$(BIN):
	$(MKDIR_P) $(BINDIR)
