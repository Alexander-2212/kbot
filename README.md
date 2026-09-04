# kbot

Telegram-бот на Go для курсу GlobalLogic DEVOPS101.
Репозиторій містить код бота, `Dockerfile`, `Makefile`, Helm-чарт і повний
CI/CD-конвеєр (GitHub Actions -> ghcr.io -> ArgoCD -> Kubernetes).

Бот: [t.me/devops101KaminskyiOV_bot](https://t.me/devops101KaminskyiOV_bot)

## Стек

- [cobra](https://github.com/spf13/cobra) - CLI-фреймворк
- [telebot.v4](https://gopkg.in/telebot.v4) - Telegram Bot API
- GitHub Actions, Jenkins - CI/CD
- ghcr.io - реєстр контейнерів
- Helm + ArgoCD - GitOps-розгортання в Kubernetes

## CI/CD

Схема автоматизованого циклу: комміт у `develop` -> збірка й публікація образу
в `ghcr.io` -> оновлення тегу в Helm-чарті -> автоматичний sync ArgoCD у кластер.

```mermaid
flowchart TD
    dev["Розробник<br/>git push"] --> develop["GitHub: гілка develop"]

    develop -->|"on: push (branches: develop)"| wf

    subgraph wf["GitHub Actions - .github/workflows/ci-cd.yml"]
        direction TB
        meta["Обчислити версію<br/>VERSION = v1.0.0-&lt;short SHA&gt;"] --> vet["make vet"]
        vet --> test["make test"]
        test --> build["make build<br/>GOOS=linux GOARCH=amd64"]
        build --> login["docker/login-action<br/>ghcr.io (GITHUB_TOKEN)"]
        login --> img["docker/build-push-action<br/>Dockerfile, platform linux/amd64"]
        img --> bump["make update-helm<br/>sed: image.tag у values.yaml"]
        bump --> commit["git commit + push<br/>ci: bump image tag [skip ci]"]
    end

    img -->|push| ghcr[("ghcr.io/alexander-2212/kbot<br/>:v1.0.0-&lt;sha&gt;-linux-amd64")]
    commit --> chart["helm/values.yaml<br/>image.tag оновлено"]

    chart -->|"polling / webhook"| argo["ArgoCD Application<br/>path: helm, revision: develop"]
    argo -->|"auto-sync, self-heal"| k8s

    subgraph k8s["Kubernetes - namespace kbot"]
        direction TB
        deploy["Deployment kbot"] --> pod["Pod kbot"]
        secret["Secret kbot-token<br/>TELE_TOKEN"] -.-> pod
    end

    ghcr -.->|"docker pull"| pod
    pod <-->|"long polling"| tg["Telegram Bot API"]
    tg <--> user["Користувач у Telegram"]
```

### Умови завдання (Модуль 6)

| Пункт | Значення |
|---|---|
| CI/CD | GitHub Actions |
| Container Registry | `ghcr.io` |
| Deploy | ArgoCD |
| Infrastructure | Kubernetes |
| Event | `push` у гілку `develop` |
| Платформа / архітектура | `linux` / `amd64` |

Результуючий образ: `ghcr.io/alexander-2212/kbot:v1.0.0-<short-sha>-linux-amd64`,
що відповідає полям `image` у [`helm/values.yaml`](helm/values.yaml):

```yaml
image:
  registry: "ghcr.io"
  repository: "alexander-2212/kbot"
  tag: "v1.0.0-<short-sha>"
  os: linux
  arch: amd64
```

### Кроки workflow

1. `checkout` гілки `develop` (повна історія - потрібен SHA для версії).
2. Обчислення версії `v1.0.0-<short SHA>` і повного посилання на образ.
3. `make vet` і `make test` - статичний аналіз і тести.
4. `make build` - крос-компіляція бінарника під `linux/amd64`.
5. Логін у `ghcr.io` вбудованим `GITHUB_TOKEN` (`packages: write`).
6. `docker/build-push-action` - збірка образу з `Dockerfile` і публікація тегу
   `v1.0.0-<sha>-linux-amd64`.
7. `make update-helm` - оновлення `image.*` у `helm/values.yaml`.
8. Комміт і `push` оновленого чарту назад у `develop`.

Комміт від `GITHUB_TOKEN` не запускає workflow повторно; додатково гілки чарту
виключені через `paths-ignore: helm/**`, тож нескінченного циклу немає.

## Jenkins Pipeline (параметризована мультиплатформенна збірка)

Декларативний pipeline - [`pipeline/jenkins.groovy`](pipeline/jenkins.groovy)
(RAW: <https://raw.githubusercontent.com/Alexander-2212/kbot/develop/pipeline/jenkins.groovy>).
Агент збірки - хост/контейнер, на якому розгорнуто Jenkins (`agent any`), тому на ньому
мають бути `go`, `make`, `git` і, для стадій `Image`/`Push`, `docker` з доступом до daemon.

### Параметри збірки

Розробник обирає параметри у формі *Build with Parameters* або запускає збірку
зі значеннями за замовчуванням.

| Параметр | Тип | За замовчуванням | Опис |
|---|---|---|---|
| `OS` | choice | `linux` | цільова ОС: `linux`, `darwin`, `windows` (GOOS) |
| `ARCH` | choice | `amd64` | цільова архітектура: `amd64`, `arm64` (GOARCH) |
| `SKIP_TESTS` | boolean | `false` | пропустити `make test` |
| `SKIP_LINT` | boolean | `false` | пропустити `gofmt` + `make vet` |
| `BUILD_IMAGE` | boolean | `false` | зібрати образ `docker buildx` (лише для `OS=linux`) |
| `PUSH_IMAGE` | boolean | `false` | запушити образ у `ghcr.io` (credentials `ghcr-credentials`) |
| `RELEASE` | string | `v1.0.0` | префікс версії, `VERSION = <RELEASE>-<short SHA>` |

### Стадії

```mermaid
flowchart LR
    co["Checkout<br/>git rev-parse -> VERSION"] --> lint["Lint<br/>gofmt, go vet<br/>(when !SKIP_LINT)"]
    lint --> test["Test<br/>go test<br/>(when !SKIP_TESTS)"]
    test --> build["Build<br/>make build BIN=kbot-OS-ARCH<br/>archiveArtifacts"]
    build --> image["Image<br/>make image<br/>(when BUILD_IMAGE && OS=linux)"]
    image --> push["Push<br/>docker login + make push<br/>(when PUSH_IMAGE)"]
```

Змінні `TARGETOS`, `TARGETARCH`, `RELEASE`, `REGISTRY`, `REPOSITORY` pipeline передає
через оточення, а `Makefile` підхоплює їх оператором `?=`, тож ті самі цілі `make`
працюють і в GitHub Actions, і в Jenkins. Бінарник отримує ім'я
`kbot-<os>-<arch>` (для Windows - з `.exe`) і зберігається як артефакт збірки.

### Локальний Jenkins

Варіант із завдання - Kind + Helm:

```bash
kind create cluster --name jenkins
helm repo add jenkinsci https://charts.jenkins.io && helm repo update
helm install jenkins jenkinsci/jenkins
kubectl port-forward svc/jenkins 8080:8080
# http://localhost:8080, пароль: kubectl get secret jenkins -o jsonpath='{.data.jenkins-admin-password}' | base64 -d
```

Варіант, що використовується в цьому репозиторії, - контейнер Jenkins, який сам є агентом
збірки: [`pipeline/jenkins/Dockerfile`](pipeline/jenkins/Dockerfile) додає до `jenkins/jenkins:lts`
Go, `make`, docker CLI/buildx та плагіни з [`plugins.txt`](pipeline/jenkins/plugins.txt),
а [`casc.yaml`](pipeline/jenkins/casc.yaml) (Configuration as Code + Job DSL) створює
адміністратора і pipeline-job `kbot`, що читає `pipeline/jenkins.groovy` з гілки `develop`.

```bash
docker build -t kbot-jenkins pipeline/jenkins
docker run -d --name kbot-jenkins -p 8081:8080 \
  -e JENKINS_ADMIN_PASSWORD=<password> \
  -v kbot-jenkins-home:/var/jenkins_home \
  -v /var/run/docker.sock:/var/run/docker.sock --group-add 0 \
  kbot-jenkins
# http://localhost:8081 -> job "kbot" -> Build with Parameters
```

Ручне створення job без JCasC: *New Item -> Pipeline -> Pipeline script from SCM*,
Repository `https://github.com/Alexander-2212/kbot.git`, Branch `*/develop`,
Script Path `pipeline/jenkins.groovy`. Після першого запуску (або *Scan*) Jenkins
зчитує блок `parameters` і кнопка *Build Now* стає *Build with Parameters*.

## Розгортання через ArgoCD

Маніфест Application - [`argocd/kbot-application.yaml`](argocd/kbot-application.yaml).

```bash
# 1. Токен бота (у git не зберігається)
kubectl create namespace kbot
kubectl -n kbot create secret generic kbot-token --from-literal=token=<TELE_TOKEN>

# 2. Application
kubectl apply -f argocd/kbot-application.yaml

# 3. Стан
kubectl -n argocd get application kbot
kubectl -n kbot get pods
kubectl -n kbot logs -l app.kubernetes.io/instance=kbot -f
# authorized as @devops101KaminskyiOV_bot, version v1.0.0-<sha>
```

ArgoCD працює в режимі `automated` + `selfHeal`, тож після комміту нового тегу
в `values.yaml` нова версія бота розкочується без ручних дій.

Якщо пакет у `ghcr.io` приватний, додайте pull-секрет:

```bash
kubectl -n kbot create secret docker-registry ghcr   --docker-server=ghcr.io --docker-username=<github-user> --docker-password=<PAT>
# і в values.yaml: imagePullSecrets: [{name: ghcr}]
```

## Локальна збірка

```bash
make help          # список цілей і поточні змінні
make image-name    # ghcr.io/alexander-2212/kbot:v1.0.0-<sha>-linux-amd64
make build         # бінарник під linux/amd64
make image         # образ локально
make image-push    # зібрати і запушити в ghcr.io
make update-helm   # оновити image.* у values.yaml
make helm-lint     # helm lint
```

Версія вшивається в бінарник через `-ldflags -X ...cmd.appVersion`, тож
`kbot version` показує ту саму версію, що й тег образу.

## Ручне встановлення чарту (без ArgoCD)

```bash
helm upgrade --install kbot helm   --namespace kbot --create-namespace   --set tele.existingSecret=kbot-token
```

Основні параметри:

| Параметр | За замовчуванням | Опис |
|---|---|---|
| `image.registry` | `ghcr.io` | реєстр образу |
| `image.repository` | `alexander-2212/kbot` | репозиторій образу |
| `image.tag` | `v1.0.0-<sha>` | тег, оновлюється CI |
| `image.os` / `image.arch` | `linux` / `amd64` | суфікс платформи в тезі |
| `tele.existingSecret` | `kbot-token` | наявний Secret із токеном |
| `tele.token` | `""` | створити Secret із чарту |
| `replicaCount` | `1` | long polling - одна репліка |

## Команди бота

| Команда | Відповідь |
|---|---|
| `hello` | привітання з іменем користувача |
| `version` | версія бота |
| `time` | поточний час сервера |
| `help` | список доступних команд |

## Локальний запуск без контейнера

```bash
export TELE_TOKEN=<your_bot_token>
go build -o kbot . && ./kbot start
```
