package kubewatch

import (
	"fmt"
	"regexp"
	"time"

	log "github.com/inconshreveable/log15"
	"github.com/kelseyhightower/envconfig"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Pod struct {
	Name      string
	Namespace string
	Status    string
	Ready     bool
}

type KubeWatch struct {
	k      K8s
	n      Notification
	logger log.Logger
}

func (kw *KubeWatch) Run(stopCh <-chan struct{}) {
	var cfg KubeWatchConfig
	if err := envconfig.Process("sindico_kube_watch", &cfg); err != nil {
		kw.logger.Error("failed to process env vars", "err", err)
		return
	}
	re, err := regexp.Compile(cfg.IgnoreNsRegexp)
	if err != nil {
		kw.logger.Error("invalid ignore regex", "err", err)
		return
	}
	kw.logger.Debug("starting")
	fn := func() {
		kw.checkCrashedPods(&cfg, re)
		kw.checkNotReadyPods(&cfg, re)
	}
	time.Sleep(5 * time.Minute)
	go wait.JitterUntil(fn, time.Duration(cfg.CircleTime)*time.Minute, 0.1, true, stopCh)
	<-stopCh
	kw.logger.Debug("stopped")
}

func (kw *KubeWatch) checkCrashedPods(cfg *KubeWatchConfig, re *regexp.Regexp) {
	podList, err := kw.k.List("")
	if err != nil {
		kw.propagateMsg(
			fmt.Sprintf(":bomb: Error on check pods status: *%v*", err),
			cfg.NotificationChannel,
		)
		return
	}

	podsInCrash := groupByNamespace(podList, re, kw.logger)
	filterCrashedsPods(podsInCrash)
	if len(podsInCrash) == 0 {
		return
	}

	msg := fmt.Sprintf(":shit: *PODS IN CRASH* on _%s_:\n\n", cfg.K8sEnv)
	for ns, pods := range podsInCrash {
		team, err := kw.k.GetLabelValue(ns, cfg.TeamNsAnnotation)
		if err != nil {
			kw.propagateMsg(
				fmt.Sprintf(":bomb: Error getting namespace label: *%v*", err),
				cfg.NotificationChannel,
			)
			return
		}
		msg = fmt.Sprintf(
			"%s*%s*: (@%s) *%d* Pod(s) in CrashLoopBackOff\n",
			msg, ns, team, len(pods))
	}

	if err = kw.propagateMsg(msg, cfg.NotificationChannel); err != nil {
		fmt.Println("Error on post msg on slack: ", err)
	}
}

func (kw *KubeWatch) checkNotReadyPods(cfg *KubeWatchConfig, re *regexp.Regexp) {
	podList, err := kw.k.List("")
	if err != nil {
		kw.propagateMsg(
			fmt.Sprintf(":bomb: Error on check pods status: *%v*", err),
			cfg.NotificationChannel,
		)
		return
	}

	podsByNamespace := groupByNamespace(podList, re, kw.logger)
	namespaceWithNotReadyPods := podsNotReadyByThreshold(podsByNamespace, cfg.NotReadyThreshold)
	if len(namespaceWithNotReadyPods) == 0 {
		return
	}

	msg := fmt.Sprintf(":warning: *NAMESPACES WITH HIGH NUMBER OF PODS NOT READY* on _%s_:\n\n", cfg.K8sEnv)
	for ns, perc := range namespaceWithNotReadyPods {
		team, err := kw.k.GetLabelValue(ns, cfg.TeamNsAnnotation)
		if err != nil {
			kw.propagateMsg(
				fmt.Sprintf(":bomb: Error getting namespace label: *%v*", err),
				cfg.NotificationChannel,
			)
		}
		msg = fmt.Sprintf("%s*%s*: (@%s) *%d %%* of pods Not Ready\n", msg, ns, team, perc)
	}

	if err = kw.propagateMsg(msg, cfg.NotificationChannel); err != nil {
		fmt.Println("Error on post msg on slack: ", err)
	}
}

func podsNotReadyByThreshold(items map[string][]Pod, threshold int) map[string]int {
	result := make(map[string]int)
	for ns, pods := range items {
		count := 0
		for _, pod := range pods {
			if pod.Status == "Running" && !pod.Ready {
				count++
			}
		}
		percNotReady := (count * 100) / len(pods)
		if percNotReady >= threshold {
			result[ns] = percNotReady
		}
	}
	return result
}

func (kw *KubeWatch) propagateMsg(msg, channel string) error {
	fmt.Println(msg)
	return kw.n.PostMessage(msg, channel)
}

func filterCrashedsPods(items map[string][]Pod) {
	for ns, pods := range items {
		podsInCrash := make([]Pod, 0)

		for _, pod := range pods {
			if pod.Status == "CrashLoopBackOff" {
				podsInCrash = append(podsInCrash, pod)
			}
		}

		if len(podsInCrash) > 0 {
			items[ns] = podsInCrash
		} else {
			delete(items, ns)
		}
	}
}

func groupByNamespace(items []Pod, re *regexp.Regexp, logger log.Logger) map[string][]Pod {
	result := make(map[string][]Pod)
	for _, pod := range items {
		if re.MatchString(pod.Namespace) {
			logger.Debug("skip ns regex", "ns", pod.Namespace)
			continue
		}
		if _, found := result[pod.Namespace]; !found {
			result[pod.Namespace] = make([]Pod, 0)
		}
		result[pod.Namespace] = append(result[pod.Namespace], pod)
	}
	return result
}

func New(k K8s, n Notification, logger log.Logger) *KubeWatch {
	return &KubeWatch{k: k, n: n, logger: logger}
}
