package slack

import (
	"net/url"
	"regexp"
)

var avatarRegex = regexp.MustCompile("^:[^:]+:$")

func getAvatarField(avatar string) string {
	if avatarRegex.MatchString(avatar) {
		return "icon_emoji"
	}
	return "icon_url"
}

func buildPayload(token, channel, username, avatar, message string) string {
	avatarField := getAvatarField(avatar)
	payload := url.Values{
		"token":     {token},
		"channel":   {channel},
		"username":  {username},
		"text":      {message},
		"as_user":   {"false"},
		"parse":     {"full"},
		avatarField: {avatar},
	}
	return payload.Encode()
}
