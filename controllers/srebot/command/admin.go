package command

import "github.com/go-chat-bot/bot"

func AdminCmd(admins map[string]bool, fn func(*bot.Cmd) (string, error)) func(*bot.Cmd) (string, error) {
	return func(command *bot.Cmd) (string, error) {
		if !admins[command.User.Nick] {
			return "You're not an admin of this bot", nil
		}
		return fn(command)
	}
}
