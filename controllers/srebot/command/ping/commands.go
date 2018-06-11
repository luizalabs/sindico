package ping

import "github.com/go-chat-bot/bot"

func ping(_ *bot.Cmd) (string, error) {
	return "pong :table_tennis_paddle_and_ball:", nil
}

func init() {
	bot.RegisterCommand("sre-ping", "Simple ping-pong check", "", ping)
}
