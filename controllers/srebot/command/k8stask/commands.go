package k8stask

import (
	"fmt"
	"strconv"

	"github.com/go-chat-bot/bot"
	"github.com/luizalabs/sindico/controllers/srebot/command"
)

type K8s interface {
	DeletePod(namespace, pod string) error
	SetReplicas(namespace, deploy string, replicas int32) error
}

type Tasks struct {
	k8s       K8s
	admins    map[string]bool
	cmdPrefix string
}

func (t *Tasks) deletePodCmd(command *bot.Cmd) (string, error) {
	if len(command.Args) < 2 {
		return "Invalid command usage", nil
	}
	ns, pod := command.Args[0], command.Args[1]
	if err := t.k8s.DeletePod(ns, pod); err != nil {
		return "", err
	}
	return "Deleted :gun:", nil
}

func (t *Tasks) setReplicas(command *bot.Cmd) (string, error) {
	if len(command.Args) < 3 {
		return "Invalid command usage", nil
	}
	ns, deploy, sReplicas := command.Args[0], command.Args[1], command.Args[2]
	replicas, err := strconv.Atoi(sReplicas)
	if err != nil {
		return "", err
	}
	if err := t.k8s.SetReplicas(ns, deploy, int32(replicas)); err != nil {
		return "", err
	}
	return fmt.Sprintf("Replicas of %s set to %s :top:", deploy, sReplicas), nil
}

func (t *Tasks) RegisterCommands() {
	bot.RegisterCommand(
		fmt.Sprintf("%s-delete-pod", t.cmdPrefix),
		"Delete a pod",
		"enter here the namespace and the name of pod to delete",
		command.AdminCmd(t.admins, t.deletePodCmd),
	)
	bot.RegisterCommand(
		fmt.Sprintf("%s-set-replicas", t.cmdPrefix),
		"Change the number of replicas of a deploy",
		"enter here the namespace, the name of deploy and desired number of replicas",
		command.AdminCmd(t.admins, t.setReplicas),
	)
}

func New(k8s K8s, cmdPrefix string, admins map[string]bool) *Tasks {
	return &Tasks{
		admins:    admins,
		k8s:       k8s,
		cmdPrefix: cmdPrefix,
	}
}
