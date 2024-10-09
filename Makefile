NAME=filecoin-utils

.PHONY: build
build:
	go build -o bin/$(NAME)