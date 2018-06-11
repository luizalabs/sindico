package notification

import (
	"github.com/drgarcia1986/slacker/slack"
)

type SlackClient interface {
	PostMessage(channel, username, avatar, msg string) error
}

type Slack struct {
	client   SlackClient
	username string
	channel  string
	avatar   string
}

func (s *Slack) PostMessage(msg string) error {
	return s.client.PostMessage(s.channel, s.username, s.avatar, msg)
}

func newSlack(cfg *Config) *Slack {
	return &Slack{
		client:   slack.New(cfg.Token),
		channel:  cfg.Channel,
		username: cfg.Username,
		avatar:   cfg.Avatar,
	}
}
