package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	telebot "gopkg.in/telebot.v4"
)

var TeleToken = os.Getenv("TELE_TOKEN")

var startCmd = &cobra.Command{
	Use:     "start",
	Aliases: []string{"kbot"},
	Short:   "Start the telegram bot",
	Run: func(cmd *cobra.Command, args []string) {
		if TeleToken == "" {
			log.Fatal("TELE_TOKEN is empty: create a bot via @BotFather and export the token")
		}

		kbot, err := telebot.NewBot(telebot.Settings{
			Token:  TeleToken,
			Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		})
		if err != nil {
			log.Fatalf("cannot create bot: %s", err)
		}

		log.Printf("authorized as @%s, version %s", kbot.Me.Username, appVersion)

		kbot.Handle(telebot.OnText, textHandler)

		kbot.Start()
	},
}

func textHandler(m telebot.Context) error {
	payload := strings.ToLower(strings.TrimSpace(m.Text()))
	log.Printf("from=%s text=%q", m.Sender().Username, payload)

	switch payload {
	case "/start", "hello":
		return m.Send(fmt.Sprintf("Hello, %s! I'm kbot %s", m.Sender().FirstName, appVersion))
	case "/version", "version":
		return m.Send(appVersion)
	case "/ping", "ping":
		return m.Send("pong")
	case "/time", "time":
		return m.Send(time.Now().Format(time.RFC1123))
	case "/help", "help":
		return m.Send("Available commands:\nhello — greeting\nping — pong\nversion — bot version\ntime — current server time\nhelp — this message")
	default:
		return m.Send(fmt.Sprintf("Unknown command: %q\nTry /help", payload))
	}
}

func init() {
	rootCmd.AddCommand(startCmd)
}
