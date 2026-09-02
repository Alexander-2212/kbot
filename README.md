# kbot

Telegram-бот на Go для курсу GlobalLogic DEVOPS101 (Модуль 2, Задача 5).

Бот: [t.me/devops101KaminskyiOV_bot](https://t.me/devops101KaminskyiOV_bot)

## Стек

- [cobra](https://github.com/spf13/cobra) - CLI-фреймворк
- [telebot.v4](https://gopkg.in/telebot.v4) - Telegram Bot API

## Встановлення

```bash
git clone https://github.com/Alexander-2212/kbot.git
cd kbot
go mod tidy
go build -o kbot .
```

## Налаштування

Створіть бота через [@BotFather](https://t.me/BotFather), отримайте токен і збережіть його у змінній оточення:

```bash
export TELE_TOKEN=<your_bot_token>
```

## Запуск

```bash
./kbot start
```

## Команди бота

| Команда | Відповідь |
|---|---|
| `hello` | привітання з іменем користувача |
| `version` | версія бота |
| `time` | поточний час сервера |
| `help` | список доступних команд |

## Команди CLI

```bash
kbot start      # запустити бота
kbot version    # вивести версію
kbot --help     # довідка
```

## Збірка через Makefile

```bash
make build                 # бінарник під поточну платформу
make all                   # linux-amd64, linux-arm64, darwin-arm64, windows-amd64
make test                  # go test ./...
make image ARCH=arm64      # образ під одну архітектуру
make image-all             # sashgun22/kbot:<version>-amd64 + -arm64
make push-all              # запушити обидва теги
```

Версія береться з `git describe --tags --always --dirty` і вшивається в бінарник
(`-ldflags -X ...cmd.appVersion`), тож `kbot version` показує версію збірки.

## Docker

Мультиархітектурний образ на базі `scratch` (~11 MB), запускається від non-root
користувача 65532:

```bash
docker build --build-arg VERSION=v1.0.1 -t kbot:v1.0.1 .
docker run --rm -e TELE_TOKEN=<token> kbot:v1.0.1
docker run --rm kbot:v1.0.1 version
```

Опубліковані образи: `sashgun22/kbot:v1.0.1-amd64`, `sashgun22/kbot:v1.0.1-arm64`.

## Kubernetes (Helm)

Чарт лежить у [`helm/kbot`](helm/kbot).

```bash
helm install kbot helm/kbot \
  --namespace kbot --create-namespace \
  --set tele.token=<TELE_TOKEN>
```

Готовий пакет чарту також доступний як asset релізу
[devops101-m05-p05 v1.0.1](https://github.com/Alexander-2212/devops101-m05-p05/releases/tag/v1.0.1):

```bash
helm install kbot https://github.com/Alexander-2212/devops101-m05-p05/releases/download/v1.0.1/kbot-1.0.1.tgz \
  --namespace kbot --create-namespace --set tele.token=<TELE_TOKEN>
```

Основні параметри:

| Параметр | За замовчуванням | Опис |
|---|---|---|
| `image.repository` | `sashgun22` | реєстр/неймспейс образу |
| `image.tag` | `v1.0.1` | тег образу |
| `image.arch` | `amd64` | суфікс архітектури: `amd64` \| `arm64` |
| `tele.token` | `""` | токен бота; чарт створює `Secret` |
| `tele.existingSecret` | `""` | використати наявний `Secret` |
| `replicaCount` | `1` | long polling — тримаємо одну репліку |

Перевірка:

```bash
kubectl -n kbot logs -l app.kubernetes.io/instance=kbot -f
# authorized as @devops101KaminskyiOV_bot, version v1.0.1
```

Видалення: `helm -n kbot uninstall kbot`.
