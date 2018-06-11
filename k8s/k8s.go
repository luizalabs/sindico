package k8s

import (
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
)

type Config struct {
	ConfigFile string `split_words:"true"`
}

type Client struct {
	clientset kubernetes.Interface
	cfg       *rest.Config
}

func New(cfg *Config) (*Client, error) {
	if cfg.ConfigFile == "" {
		return newInClusterK8sClient()
	}
	return newOutOfClusterK8sClient(cfg)
}
