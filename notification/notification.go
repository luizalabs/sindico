package notification

type Config struct {
	Channel  string `split_words:"true" default:"#alerts"`
	Avatar   string `split_words:"true"`
	Token    string `split_words:"true"`
	Username string `split_words:"true" default:"sindico"`
}

type Poster interface {
	PostMessage(msg string) error
}

type Client struct {
	Poster
}

func New(cfg *Config) *Client {
	return &Client{newSlack(cfg)}
}
