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
