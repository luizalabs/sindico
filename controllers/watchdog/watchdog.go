package watchdog

import (
	log "github.com/inconshreveable/log15"

	"k8s.io/client-go/kubernetes"
)

type K8s interface {
	NewClientset() (kubernetes.Interface, error)
}

type SubController interface {
	Run(done <-chan struct{})
}

type Controller struct {
	subCtrls []SubController
}

func NewController(k8s K8s) *Controller {
	subCtrls := []SubController{
		newHPASubController(k8s, log.New("subcontroller", "hpa")),
		newLimitsSubController(k8s, log.New("subcontroller", "limits")),
	}
	return &Controller{subCtrls: subCtrls}
}

func (c *Controller) Run(stopCh <-chan struct{}) {
	for _, subCtrl := range c.subCtrls {
		go subCtrl.Run(stopCh)
	}
}
