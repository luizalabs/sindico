package notification

type Config struct {
	Avatar   string `split_words:"true"`
	Token    string `split_words:"true"`
	Username string `split_words:"true" default:"sindico"`
}

type Poster interface {
	PostMessage(msg, channel string) error
}

type Client struct {
	Poster
}

func New(cfg *Config) *Client {
	return &Client{newSlack(cfg)}
}
