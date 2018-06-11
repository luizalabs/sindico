package keeptrack

import (
	"bytes"
	"fmt"
	"io"

	"github.com/go-chat-bot/bot"
	"github.com/luizalabs/sindico/controllers/srebot/command"
)

const (
	keepTrackCrashsCommand = "sre-track-crashs"
	keepTrackTasksCommand  = "sre-track-tasks"
	keepTrackCountCommand  = "sre-track-count"
)

type KeepTrack struct {
	admins       map[string]bool
	taskStorage  map[string]int
	crashStorage map[string]int
}

func (kt *KeepTrack) countCmd(_ *bot.Cmd) (string, error) {
	if len(kt.taskStorage) == 0 && len(kt.crashStorage) == 0 {
		return "nothing here", nil
	}

	b := new(bytes.Buffer)
	printCount(b, "Tasks", kt.taskStorage)
	printCount(b, "Crashed pods", kt.crashStorage)

	return b.String(), nil
}

func (kt *KeepTrack) trackCmd(command *bot.Cmd) (string, error) {
	if command.Command == keepTrackCrashsCommand {
		track(kt.crashStorage, command.Args)
	} else {
		track(kt.taskStorage, command.Args)
	}
	return "Registered!", nil
}

func track(storage map[string]int, args []string) {
	for _, arg := range args {
		storage[arg]++
	}
}

func printCount(w io.Writer, label string, storage map[string]int) {
	if len(storage) == 0 {
		return
	}

	fmt.Fprintf(w, "*%s*\n```\n", label)
	for k, v := range storage {
		fmt.Fprintf(w, "%s\t%d\n", k, v)
	}
	fmt.Fprintln(w, "```")
}

func (kt *KeepTrack) RegisterCommands() {
	bot.RegisterCommand(
		keepTrackCrashsCommand,
		"Applications with crashed pods",
		"enter here the namespace and number of crashed pods",
		command.AdminCmd(kt.admins, kt.trackCmd),
	)
	bot.RegisterCommand(
		keepTrackTasksCommand,
		"Annoying task was done by hand",
		"enter here the name of task",
		command.AdminCmd(kt.admins, kt.trackCmd),
	)
	bot.RegisterCommand(
		keepTrackCountCommand,
		"Keep track count",
		"",
		command.AdminCmd(kt.admins, kt.countCmd),
	)
}

func New(admins map[string]bool) *KeepTrack {
	kt := &KeepTrack{
		admins:       admins,
		taskStorage:  make(map[string]int),
		crashStorage: make(map[string]int),
	}
	return kt
}
