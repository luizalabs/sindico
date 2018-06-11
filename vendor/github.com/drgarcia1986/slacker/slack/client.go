/*
	Package slack implements a Slack integartion.

	Usage:

	Create a new client with integration token:

		client := slack.New("slack-integration-token")

	And post a message:

		client.PostMessage(channel, username, avatar, message)
*/
package slack

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
)

type Client struct {
	Token string
}

type slackResponse struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
}

var slackUrl = "https://slack.com/api/chat.postMessage"

// PostMessage post a message in slack organization defined by token integration
// on a specified channel.
// The avatar can be a url or an emoji (e.g. :scream:)
func (c *Client) PostMessage(channel, username, avatar, message string) error {
	payload := buildPayload(c.Token, channel, username, avatar, message)
	data := bytes.NewBufferString(payload)

	resp, err := http.Post(slackUrl, "application/x-www-form-urlencoded", data)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var response slackResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}
	if !response.Ok {
		return errors.New(response.Error)
	}
	return nil
}

func New(token string) *Client {
	return &Client{Token: token}
}
