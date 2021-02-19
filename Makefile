# Makefile - use `make help` to get a list of possible commands.
#
# Note - Comments inside this makefile should be made using a single
# hashtag `#`, lines with double hash-tags will be the messages that
# are printed with during the `make help` command.

# Setting up some common variables.
PROJECTNAME=$(shell basename "$(PWD)")

# Go related variable(s)
GOFILES=$(wildcard *.go)

# Redirecting error output to a file, can be displayed it during development.
STDERR=/tmp/$(PROJECTNAME)-stderr.txt

# Setting `help` to be the default goal of the make file - ensures firing a blank
# `make` command will print help.
.DEFAULT_GOAL := help

# Make is verbose in Linux. Make it silent.
MAKEFLAGS += --silent

.PHONY: help
help: Makefile
	@echo
	@echo " Commands available in "$(PROJECTNAME)":"
	@echo
	@sed -n 's/^[ \t]*##//p' $< | column -t -s ':' |  sed -e 's/^//'
	@echo

.PHONY: run
run:
	## `run`: Run the main project file. Pass arguments as `make run q="--log"`
	go run main.go $(q)

# Will install missing dependencies
.PHONY: install
install:
	@echo "  >  Getting dependency..."
	go get -v $(get)
	go mod tidy

.PHONY: local-setup
local-setup:
	## `local-setup`: Setup development environment locally
	@echo "Setting up pre-commit"
	pip install pre-commit
	pre-commit install
	@echo "Installing testing environment"
	bash ./setup.sh

.PHONY: codestyle
codestyle:
	## `codestyle`: Auto-Format code using GoFmt and GoImports
	@echo -e "\n\t> Running GoFmt"
	@gofmt -l $(GOFILES)
	@echo -e "\n\t> Running GoImports"
	@./tmp/goimports -w -l $(GOFILES)

.PHONY: checkstyle
checkstyle:
	## `checkstyle`: Run linter(s) and check code-style
	@echo -e "\n\t> Running GoVet"
	@go vet $(GOFILES)
	@echo -e "\t> Running Shadow"
	@go vet --vettool=./tmp/shadow
	@echo -e "\t> Running GoImports"
	@./tmp/goimports -e -l $(GOFILES)
	@echo -e "\t> Running StaticCheck"
	@./tmp/staticcheck ./...
	@echo -e "\t> Running ErrCheck"
	@./tmp/errcheck -abspath -asserts -blank
	@echo -e "\t> Running Golang CI - Lint"
	@./tmp/golangci-lint run


.PHONY: test
test:
	## `test`: Run tests and generate coverage report
	go test ./... -race -covermode=atomic -coverprofile=./coverage/coverage.txt -gcflags=-l
	go tool cover -html=./coverage/coverage.txt -o ./coverage/coverage.html

.PHONY: test-suite
## `test-suite`: Check-styles and run tests with a single command
test-suite: checkstyle test

go-clean:
	@echo "  >  Cleaning build cache"
	go clean
