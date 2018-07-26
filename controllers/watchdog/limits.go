package watchdog

import (
	"regexp"
	"time"

	log "github.com/inconshreveable/log15"
	"github.com/kelseyhightower/envconfig"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type LimitsSubController struct {
	k8s    K8s
	logger log.Logger
}

type LimitsSubControllerConfig struct {
	Interval       time.Duration `split_words:"true" default:"5m"`
	IgnoreNsRegexp string        `split_words:"true" default:"nginx-.+|sindico|default|kube-.+"`
	RequestCPU     string        `split_words:"true" default:"100m"`
	RequestMemory  string        `split_words:"true" default:"512Mi"`
}

func (l *LimitsSubController) Run(stopCh <-chan struct{}) {
	var cfg LimitsSubControllerConfig
	if err := envconfig.Process("sindico_watchdog_limits", &cfg); err != nil {
		l.logger.Error("failed to process env vars", "err", err)
		return
	}
	re, err := regexp.Compile(cfg.IgnoreNsRegexp)
	if err != nil {
		l.logger.Error("invalid ignore regex", "err", err)
		return
	}
	l.logger.Debug("starting")
	fn := func() { l.run(re, &cfg) }
	go wait.JitterUntil(fn, cfg.Interval, 0.1, true, stopCh)
	<-stopCh
	l.logger.Debug("stopped")
}

func (l *LimitsSubController) run(re *regexp.Regexp, cfg *LimitsSubControllerConfig) {
	cli, err := l.k8s.NewClientset()
	if err != nil {
		l.logger.Error("failed to build clientset", "err", err)
		return
	}
	lims, err := cli.CoreV1().LimitRanges(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		l.logger.Error("limits list failed", "err", err)
		return
	}
	for _, lim := range lims.Items {
		ns := lim.ObjectMeta.Namespace
		if re != nil && re.MatchString(ns) {
			l.logger.Debug("skip ns regex", "ns", ns)
			continue
		}
		l.updateLimits(cfg, &lim)
	}
}

func (l *LimitsSubController) updateLimits(cfg *LimitsSubControllerConfig, lim *k8sv1.LimitRange) {
	ns := lim.ObjectMeta.Namespace
	if len(lim.Spec.Limits) == 0 {
		l.logger.Debug("request not found", "ns", ns)
		return
	}
	defReq := lim.Spec.Limits[0].DefaultRequest
	cpu := defReq.Cpu()
	mem := defReq.Memory()
	cfgCPU, _ := resource.ParseQuantity(cfg.RequestCPU)
	cfgMem, _ := resource.ParseQuantity(cfg.RequestMemory)
	var update bool
	if cpu != nil && cpu.Cmp(cfgCPU) > 0 {
		l.logger.Debug("set cpu", "ns", ns, "cpu", cfgCPU.String())
		defReq["cpu"] = cfgCPU
		update = true
	}
	if mem != nil && mem.Cmp(cfgMem) > 0 {
		l.logger.Debug("set mem", "ns", ns, "mem", cfgMem.String())
		defReq["memory"] = cfgMem
		update = true
	}
	if update {
		cli, err := l.k8s.NewClientset()
		if err != nil {
			l.logger.Error("failed to build clientset", "ns", ns, "err", err)
		}
		if _, err := cli.CoreV1().LimitRanges(ns).Update(lim); err != nil {
			l.logger.Error("failed to update limits", "ns", ns, "err", err)
		}
	} else {
		l.logger.Debug("skipped update", "ns", ns)
	}
}

func newLimitsSubController(k8s K8s, logger log.Logger) *LimitsSubController {
	return &LimitsSubController{k8s: k8s, logger: logger}
}
