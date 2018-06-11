package srebot

import (
	"strings"

	"github.com/go-chat-bot/bot/slack"
	log "github.com/inconshreveable/log15"
	"github.com/kelseyhightower/envconfig"
	"github.com/luizalabs/sindico/controllers/srebot/command/k8stask"
	"github.com/luizalabs/sindico/controllers/srebot/command/keeptrack"
	_ "github.com/luizalabs/sindico/controllers/srebot/command/ping"
)

type K8s interface {
	DeletePod(namespace, pod string) error
	SetReplicas(namespace, deploy string, replicas int32) error
}

type Controller struct {
	k8s    K8s
	logger log.Logger
}

type SreBotConfig struct {
	SlackToken string `split_words:"true" default:""`
	Admins     string `split_words:"true" default:"admin"`
	CmdPrefix  string `split_words:"true" default:"production"`
}

func (c *Controller) Run(stopCh <-chan struct{}) {
	defer c.logger.Debug("stopped")
	var cfg SreBotConfig
	if err := envconfig.Process("sindico_sre_bot", &cfg); err != nil {
		log.Error("failed to process env vars", "err", err)
		return
	}
	c.logger.Debug("starting")
	admins := make(map[string]bool)
	for _, a := range strings.Split(cfg.Admins, ",") {
		admins[a] = true
	}
	keeptrack.New(admins).RegisterCommands()
	k8stask.New(c.k8s, cfg.CmdPrefix, admins).RegisterCommands()
	slack.Run(cfg.SlackToken)
}

func NewController(k8s K8s) *Controller {
	logger := log.New("controller", "srebot")
	return &Controller{k8s: k8s, logger: logger}
}
