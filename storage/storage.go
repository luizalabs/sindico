package storage

import "io"

type Config struct {
	Key    string `split_words:"true"`
	Secret string `split_words:"true"`
	Region string `split_words:"true" default:"us-east-1"`
	Bucket string `split_words:"true" default:"sindico"`
}

type Uploader interface {
	UploadFile(path string, r io.ReadSeeker) error
}

type Client struct {
	Uploader
}

func New(cfg *Config) *Client {
	return &Client{newS3(cfg)}
}
