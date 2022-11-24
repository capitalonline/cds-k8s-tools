MAIN_FILE=$(MAIN_FILE)
BIN_FILE=$(BIN_FILE)
IMAGE?=registry-bj.capitalonline.net/cck/${BIN_FILE}
IMAGE_OVERSEA=capitalonline/${BIN_FILE}
VERSION=v2.0.4
GIT_COMMIT?=$(shell git rev-parse HEAD)
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS?=" -s -w"
RELEASES_PATH=./releases/
SNAT_RELEASE_FILE=${RELEASES_PATH}/cds-snat-configuration.yaml
.EXPORT_ALL_VARIABLES:

.PHONY: build
build:
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags ${LDFLAGS} -o bin/${BIN_FILE} ./cmd/${MAIN_FILE}

.PHONY: container-binary
container-binary:
	CGO_ENABLED=0 GOARCH="amd64" GOOS="linux" go build -ldflags ${LDFLAGS} -o bin/${BIN_FILE} ./cmd/${MAIN_FILE}

.PHONY: image-release
image-release:
	docker build -t $(IMAGE):$(VERSION) .
	docker tag $(IMAGE):$(VERSION) $(IMAGE_OVERSEA):$(VERSION)

.PHONY: image
image:
	docker build -t $(IMAGE):latest .

.PHONY: release
release: image-release
	docker push $(IMAGE):$(VERSION)
	docker push $(IMAGE_OVERSEA):$(VERSION)

.PHONY: unit-test
unit-test:
	@echo "**************************** running unit test ****************************"
	go test -v -race ./pkg/...
