package watchdog

import (
	"regexp"
	"time"

	log "github.com/inconshreveable/log15"
	"github.com/kelseyhightower/envconfig"

	asv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type HPASubController struct {
	logger log.Logger
	k8s    K8s
}

type HPASubControllerConfig struct {
	Interval       time.Duration `split_words:"true" default:"5m"`
	IgnoreNsRegexp string        `split_words:"true" default:"nginx-.+|sindico|default|kube-.+"`
	MinReplicas    int32         `split_words:"true" default:"2"`
	MaxReplicas    int32         `split_words:"true" default:"2"`
}

func (h *HPASubController) Run(stopCh <-chan struct{}) {
	var cfg HPASubControllerConfig
	if err := envconfig.Process("sindico_watchdog_hpa", &cfg); err != nil {
		h.logger.Error("failed to process env vars", "err", err)
		return
	}
	re, err := regexp.Compile(cfg.IgnoreNsRegexp)
	if err != nil {
		h.logger.Error("invalid ignore regex", "err", err)
		return
	}
	h.logger.Debug("starting")
	fn := func() { h.run(re, &cfg) }
	go wait.JitterUntil(fn, cfg.Interval, 0.1, true, stopCh)
	<-stopCh
	h.logger.Debug("stopped")
}

func (h *HPASubController) run(re *regexp.Regexp, cfg *HPASubControllerConfig) {
	cli, err := h.k8s.NewClientset()
	if err != nil {
		h.logger.Error("failed to build clientset", "err", err)
		return
	}
	hpas, err := cli.AutoscalingV1().HorizontalPodAutoscalers(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		h.logger.Error("hpa list failed", "err", err)
		return
	}
	for _, hpa := range hpas.Items {
		ns := hpa.ObjectMeta.Namespace
		if re != nil && re.MatchString(ns) {
			h.logger.Debug("skip ns regex", "ns", ns)
			continue
		}
		h.updateHPA(cfg, &hpa)
	}
}

func (h *HPASubController) updateHPA(cfg *HPASubControllerConfig, hpa *asv1.HorizontalPodAutoscaler) {
	oldSpec := hpa.Spec
	min := cfg.MinReplicas
	ns := hpa.ObjectMeta.Namespace
	if hpa.Spec.MinReplicas != nil && *hpa.Spec.MinReplicas > min {
		h.logger.Debug("set min replicas", "ns", ns, "min", min)
		hpa.Spec.MinReplicas = &min
	}
	if hpa.Spec.MaxReplicas > cfg.MaxReplicas {
		h.logger.Debug("set max replicas", "ns", ns, "max", cfg.MaxReplicas)
		hpa.Spec.MaxReplicas = cfg.MaxReplicas
	}
	if hpa.Spec == oldSpec {
		h.logger.Debug("skipped update", "ns", ns)
	} else {
		cli, err := h.k8s.NewClientset()
		if err != nil {
			h.logger.Error("failed to build client set", "err", err)
			return
		}
		if _, err := cli.AutoscalingV1().HorizontalPodAutoscalers(ns).Update(hpa); err != nil {
			h.logger.Error("failed to update hpa", "err", err)
		}
	}
}

func newHPASubController(k8s K8s, logger log.Logger) *HPASubController {
	return &HPASubController{k8s: k8s, logger: logger}
}
