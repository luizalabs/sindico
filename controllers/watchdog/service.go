package watchdog

import (
	"fmt"
	"regexp"
	"time"

	log "github.com/inconshreveable/log15"
	"github.com/kelseyhightower/envconfig"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Notification interface {
	PostMessage(msg, channel string) error
}

type ServiceSubController struct {
	logger log.Logger
	k8s    K8s
	nt     Notification
}

type ServiceSubControllerConfig struct {
	Interval                  time.Duration `split_words:"true" default:"5m"`
	CheckFirewall             bool          `split_words:"true" default:"false"`
	CheckFirewallSkipNsRegexp string        `split_words:"true" default:"nginx-.+|sindico|default|kube-.+"`
	NotificationChannel       string        `split_words:"true" default:"#alerts"`
	TeamNsLabel               string        `split_words:"true" default:"teresa.io/team"`
}

func (s *ServiceSubController) Run(stopCh <-chan struct{}) {
	var cfg ServiceSubControllerConfig
	if err := envconfig.Process("sindico_watchdog_service", &cfg); err != nil {
		s.logger.Error("failed to process env vars", "err", err)
		return
	}
	re, err := regexp.Compile(cfg.CheckFirewallSkipNsRegexp)
	if err != nil {
		s.logger.Error("invalid check firewall skip regex", "err", err)
		return
	}
	s.logger.Debug("starting")
	fn := func() { s.checkFirewall(re, &cfg) }
	go wait.JitterUntil(fn, cfg.Interval, 0.1, true, stopCh)
	<-stopCh
	s.logger.Debug("stopped")
}

func (s *ServiceSubController) checkFirewall(re *regexp.Regexp, cfg *ServiceSubControllerConfig) {
	if !cfg.CheckFirewall {
		s.logger.Debug("firewall check disabled")
		return
	}
	cli, err := s.k8s.NewClientset()
	if err != nil {
		s.logger.Error("failed to build clientset", "err", err)
		return
	}
	svcs, err := cli.CoreV1().Services(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		s.logger.Error("services list failed", "err", err)
		return
	}
	for _, svc := range svcs.Items {
		ns := svc.ObjectMeta.Namespace
		if re != nil && re.MatchString(ns) {
			s.logger.Debug("skip ns regex", "ns", ns)
			continue
		}
		s.checkService(cfg, &svc)
	}
}

func (s *ServiceSubController) checkService(cfg *ServiceSubControllerConfig, svc *k8sv1.Service) {
	if len(svc.Spec.LoadBalancerSourceRanges) == 0 && svc.Spec.Type == "LoadBalancer" {
		ns := svc.Namespace
		team, err := s.k8s.GetLabelValue(ns, cfg.TeamNsLabel)
		if err != nil {
			s.notify(
				fmt.Sprintf(":bomb: failed to get namespace label: *%v*", err),
				cfg.NotificationChannel,
			)
			return
		}
		s.notify(
			fmt.Sprintf(":shit: (@%s) namespace *%s* without firewall rules", team, ns),
			cfg.NotificationChannel,
		)
	}
}

func (s *ServiceSubController) notify(msg, channel string) {
	if err := s.nt.PostMessage(msg, channel); err != nil {
		s.logger.Error("failed to post message", "err", err)
	}
}

func newServiceSubController(k8s K8s, logger log.Logger, nt Notification) *ServiceSubController {
	return &ServiceSubController{k8s: k8s, logger: logger, nt: nt}
}
