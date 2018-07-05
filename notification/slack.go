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
	avatar   string
}

func (s *Slack) PostMessage(msg, channel string) error {
	return s.client.PostMessage(channel, s.username, s.avatar, msg)
}

func newSlack(cfg *Config) *Slack {
	return &Slack{
		client:   slack.New(cfg.Token),
		username: cfg.Username,
		avatar:   cfg.Avatar,
	}
}
