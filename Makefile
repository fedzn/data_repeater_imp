BINARY_NAME=data_repeater
OS=linux
ARCH=amd64
build:
	GOOS=$(OS) GOARCH=$(ARCH) go build -o ./bin/$(BINARY_NAME)_$(OS)_$(ARCH) ./data_repeater.go
