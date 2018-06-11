package kubewatch

import (
	log "github.com/inconshreveable/log15"
)

type Controller struct {
	*KubeWatch
}

type Notification interface {
	PostMessage(msg string) error
}

type K8s interface {
	List(namespace string) ([]Pod, error)
	GetLabelValue(namespace, label string) (string, error)
}

type KubeWatchConfig struct {
	CircleTime        int    `split_words:"true" default:"5"`
	K8sEnv            string `split_words:"true" default:"production"`
	NotReadyThreshold int    `split_words:"true" default:"60"`
	IgnoreNsRegexp    string `split_words:"true" default:"default"`
	TeamNsAnnotation  string `split_words:"true" default:"teresa.io/team"`
}

func NewController(k8s K8s, nt Notification) *Controller {
	logger := log.New("controller", "kubewatch")
	return &Controller{KubeWatch: New(k8s, nt, logger)}
}
