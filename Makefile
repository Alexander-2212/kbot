APP        ?= kbot
REGISTRY   ?= sashgun22
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
TARGETOS   ?= linux
ARCH       ?= amd64
ARCHS      ?= amd64 arm64
CHART_DIR  ?= helm/kbot
DIST       ?= dist

.PHONY: help format test build linux arm macos windows all image image-all push push-all helm-lint helm-package clean

help:
	@echo "Збірка:    build linux arm macos windows all"
	@echo "Якість:    format test"
	@echo "Образи:    image [ARCH=amd64|arm64] image-all push push-all"
	@echo "Helm:      helm-lint helm-package"
	@echo "Змінні:    APP=$(APP) REGISTRY=$(REGISTRY) VERSION=$(VERSION) ARCHS='$(ARCHS)'"

format:
	gofmt -s -w .

test:
	go test ./...

# Бінарник під поточну платформу
build:
	CGO_ENABLED=0 go build -trimpath \
		-ldflags "-s -w -X github.com/Alexander-2212/kbot/cmd.appVersion=$(VERSION)" \
		-o $(APP) .

linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
		-ldflags "-s -w -X github.com/Alexander-2212/kbot/cmd.appVersion=$(VERSION)" \
		-o $(APP)-linux-amd64 .

arm:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath \
		-ldflags "-s -w -X github.com/Alexander-2212/kbot/cmd.appVersion=$(VERSION)" \
		-o $(APP)-linux-arm64 .

macos:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath \
		-ldflags "-s -w -X github.com/Alexander-2212/kbot/cmd.appVersion=$(VERSION)" \
		-o $(APP)-darwin-arm64 .

windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath \
		-ldflags "-s -w -X github.com/Alexander-2212/kbot/cmd.appVersion=$(VERSION)" \
		-o $(APP)-windows-amd64.exe .

all: linux arm macos windows

# Образ під одну архітектуру: make image ARCH=arm64
image:
	docker buildx build \
		--platform $(TARGETOS)/$(ARCH) \
		--build-arg VERSION=$(VERSION) \
		-t $(REGISTRY)/$(APP):$(VERSION)-$(ARCH) \
		--load .

image-all:
	@for a in $(ARCHS); do $(MAKE) image ARCH=$$a || exit 1; done

push:
	docker push $(REGISTRY)/$(APP):$(VERSION)-$(ARCH)

push-all:
	@for a in $(ARCHS); do $(MAKE) push ARCH=$$a || exit 1; done

helm-lint:
	helm lint $(CHART_DIR) --set tele.token=dummy

helm-package: helm-lint
	mkdir -p $(DIST)
	helm package $(CHART_DIR) -d $(DIST)

clean:
	rm -f $(APP) $(APP)-*
	rm -rf $(DIST)
	-docker rmi $(foreach a,$(ARCHS),$(REGISTRY)/$(APP):$(VERSION)-$(a))
