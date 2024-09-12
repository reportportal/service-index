.DEFAULT_GOAL := build

COMMIT_HASH = `git rev-parse --short HEAD 2>/dev/null`
BUILD_DATE = `date +%FT%T%z`

GO = go
BINARY_DIR=bin
RELEASE_DIR=release

BUILD_DEPS:= github.com/avarabyeu/releaser@master
GODIRS_NOVENDOR = $(shell go list ./... | grep -v /vendor/)
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")
PWD = $(shell pwd)
PACKAGE_COMMONS=github.com/reportportal/service-index
REPO_NAME=reportportal/service-index

BUILD_INFO_LDFLAGS=-ldflags "-extldflags '"-static"' -X ${PACKAGE_COMMONS}/buildinfo.repo=${REPO_NAME} -X ${PACKAGE_COMMONS}/buildinfo.branch=${COMMIT_HASH} -X ${PACKAGE_COMMONS}/buildinfo.buildDate=${BUILD_DATE} -X ${PACKAGE_COMMONS}/buildinfo.version=${v}"
IMAGE_NAME=reportportal-dev/service-index$(IMAGE_POSTFIX)

.PHONY: get-build-deps vendor test build

help:
	@echo "build      - go build"
	@echo "test       - go test"
	@echo "checkstyle - gofmt+golint+misspell"

get-build-deps:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.61.0
	$(GO) install $(BUILD_DEPS)

test:
	ls -la
	$(GO) test ${GODIRS_NOVENDOR}


checkstyle:
	./bin/golangci-lint run --fast --disable=errcheck --disable=gas

lint: checkstyle

fmt:
	gofmt -l -w -s ${GOFILES_NOVENDOR}
	gofumpt -l -w ${GOFILES_NOVENDOR}
	goimports -local github.com/reportportal/service-index -w ${GOFILES_NOVENDOR}
	gci write --skip-generated --section Standard --section Default --section "Prefix(github.com/reportportal/service-index)" ${GOFILES_NOVENDOR}

# Builds server
build:
	CGO_ENABLED=0 GOOS=linux $(GO) build ${BUILD_INFO_LDFLAGS} -o ${BINARY_DIR}/service-index ./
	#CGO_ENABLED=0 $(GO) build ${BUILD_INFO_LDFLAGS} -o ${BINARY_DIR}/service-index ./


# Builds server
build-release: test checkstyle
	$(eval v := $(shell releaser bump))
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux $(GO) build ${BUILD_INFO_LDFLAGS} -o ${RELEASE_DIR}/service-index_linux_amd64 ./
	CGO_ENABLED=0 GOARCH=amd64 GOOS=windows $(GO) build ${BUILD_INFO_LDFLAGS} -o ${RELEASE_DIR}/service-index_win_amd64.exe ./
	#gox -output "release/{{.Dir}}_{{.OS}}_{{.Arch}}" -os "linux windows" -arch "amd64" ${BUILD_INFO_LDFLAGS}
	chmod -R +xr release

# Builds the container
build-image:
	docker build --build-arg version=$(v) -t "$(IMAGE_NAME)" -f Dockerfile-develop .

release: get-build-deps build-release
	releaser release --bintray.token ${BINTRAY_TOKEN}

# Builds the container and pushes to private registry
pushDev:
	echo "Registry is not provided"
	if [ -d ${REGISTRY} ] ; then echo "Provide registry"; exit 1 ; fi
	docker tag "$(IMAGE_NAME)" "$(REGISTRY)/$(IMAGE_NAME):latest"
	docker push "$(REGISTRY)/$(IMAGE_NAME):latest"

clean:
	if [ -d ${BINARY_DIR} ] ; then rm -r ${BINARY_DIR} ; fi
	if [ -d 'build' ] ; then rm -r 'build' ; fi
