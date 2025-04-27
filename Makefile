GOCMD			:=$(shell which go)
GOBUILD			:=$(GOCMD) build
StripGoPath     :=-trimpath


IMPORT_PATH		:=image_guard/cmd
BUILD_TIME		:=$(shell date "+%F %T")
COMMIT_ID       :=$(shell git rev-parse HEAD)
GO_VERSION      :=$(shell $(GOCMD) version)
VERSION			:=$(shell git describe --tags)
BUILD_USER		:=$(shell whoami)
FLAG			:="-X '${IMPORT_PATH}.buildTime=${BUILD_TIME}' -X '${IMPORT_PATH}.commitID=${COMMIT_ID}' -X '${IMPORT_PATH}.goVersion=${GO_VERSION}' -X '${IMPORT_PATH}.goVersion=${GO_VERSION}' -X '${IMPORT_PATH}.Version=${VERSION}' -X '${IMPORT_PATH}.buildUser=${BUILD_USER}'"

BINARY_DIR=bin
BINARY_NAME:=image_guard



build:
	@CGO_ENABLED=0 $(GOBUILD) $(StripGoPath) -ldflags $(FLAG) -o $(BINARY_DIR)/$(BINARY_NAME)

# linux
build-linux:
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(StripGoPath) -ldflags $(FLAG) -o $(BINARY_DIR)/$(BINARY_NAME)-$(VERSION)-linux


#mac
build-darwin:
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(StripGoPath) -ldflags $(FLAG) -o $(BINARY_DIR)/$(BINARY_NAME)-$(VERSION)-darwin


# 全平台
build-all:
	make build-linux
	#cd bin&&tar zcf ${BINARY_NAME}.tgz ${BINARY_NAME}

check:
	@golangci-lint run ./...