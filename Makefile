APP        ?= kbot
# Ім'я вихідного бінарника (Jenkins передає kbot-<os>-<arch>[.exe])
BIN        ?= $(APP)
# Реєстр і репозиторій образу: ghcr.io/<owner>/<app> (ghcr вимагає нижній регістр)
REGISTRY   ?= ghcr.io
OWNER      ?= alexander-2212
REPOSITORY ?= $(OWNER)/$(APP)

# Версія = <реліз>-<короткий SHA коміту>, напр. v1.0.0-106879e
RELEASE    ?= v1.0.0
COMMIT     ?= $(shell git rev-parse --short=7 HEAD 2>/dev/null || echo dev)
VERSION    ?= $(RELEASE)-$(COMMIT)

# Цільова платформа
TARGETOS   ?= linux
TARGETARCH ?= amd64

# Повне посилання на образ: ghcr.io/alexander-2212/kbot:v1.0.0-106879e-linux-amd64
IMAGE       = $(REGISTRY)/$(REPOSITORY):$(VERSION)-$(TARGETOS)-$(TARGETARCH)

CHART_DIR  ?= helm
VALUES     ?= $(CHART_DIR)/values.yaml
DIST       ?= dist

.PHONY: help format vet test build image push image-push update-helm helm-lint helm-package image-name clean

help:
	@echo "Якість:    format vet test"
	@echo "Збірка:    build image push image-push"
	@echo "Helm:      update-helm helm-lint helm-package"
	@echo "Сервісне:  image-name clean"
	@echo "Змінні:    APP=$(APP) REGISTRY=$(REGISTRY) REPOSITORY=$(REPOSITORY)"
	@echo "           VERSION=$(VERSION) TARGETOS=$(TARGETOS) TARGETARCH=$(TARGETARCH) BIN=$(BIN)"
	@echo "Образ:     $(IMAGE)"

# Вивести посилання на образ (використовується у CI та як відповідь на завдання)
image-name:
	@echo $(IMAGE)

format:
	gofmt -s -w .

vet:
	go vet ./...

test:
	go test ./...

# Бінарник під цільову платформу
build:
	CGO_ENABLED=0 GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -trimpath \
		-ldflags "-s -w -X github.com/Alexander-2212/kbot/cmd.appVersion=$(VERSION)" \
		-o $(BIN) .

# Образ під цільову платформу
image:
	docker buildx build \
		--platform $(TARGETOS)/$(TARGETARCH) \
		--build-arg TARGETOS=$(TARGETOS) \
		--build-arg TARGETARCH=$(TARGETARCH) \
		--build-arg VERSION=$(VERSION) \
		-t $(IMAGE) \
		--load .

push:
	docker push $(IMAGE)

# Зібрати і одразу запушити (у CI зручніше одним кроком)
image-push:
	docker buildx build \
		--platform $(TARGETOS)/$(TARGETARCH) \
		--build-arg TARGETOS=$(TARGETOS) \
		--build-arg TARGETARCH=$(TARGETARCH) \
		--build-arg VERSION=$(VERSION) \
		-t $(IMAGE) \
		--push .

# Оновити тег/координати образу в helm-чарті - це і є GitOps-комміт для ArgoCD
update-helm:
	sed -i -e "s|^  registry: .*|  registry: \"$(REGISTRY)\"|" \
	       -e "s|^  repository: .*|  repository: \"$(REPOSITORY)\"|" \
	       -e "s|^  tag: .*|  tag: \"$(VERSION)\"|" \
	       -e "s|^  os: .*|  os: $(TARGETOS)|" \
	       -e "s|^  arch: .*|  arch: $(TARGETARCH)|" \
	       $(VALUES)
	@echo "оновлено $(VALUES) -> $(IMAGE)"

helm-lint:
	helm lint $(CHART_DIR) --set tele.token=dummy

helm-package: helm-lint
	mkdir -p $(DIST)
	helm package $(CHART_DIR) -d $(DIST)

clean:
	rm -f $(APP) $(APP)-* $(BIN)
	rm -rf $(DIST)
	-docker rmi $(IMAGE)
