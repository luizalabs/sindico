package manager

import (
	"context"

	log "github.com/inconshreveable/log15"
	"github.com/kelseyhightower/envconfig"
	"github.com/luizalabs/sindico/controllers/etcdbackup"
	"github.com/luizalabs/sindico/controllers/kubewatch"
	"github.com/luizalabs/sindico/controllers/srebot"
	"github.com/luizalabs/sindico/controllers/watchdog"
	"github.com/luizalabs/sindico/k8s"
	"github.com/luizalabs/sindico/notification"
	"github.com/luizalabs/sindico/storage"
	"github.com/pkg/errors"
)

type Controller interface {
	Run(stopCh <-chan struct{})
}

func newK8s() (*k8s.Client, error) {
	var cfg k8s.Config
	if err := envconfig.Process("sindico_k8s", &cfg); err != nil {
		return nil, err
	}
	return k8s.New(&cfg)
}

func newStorage() (*storage.Client, error) {
	var cfg storage.Config
	if err := envconfig.Process("sindico_storage", &cfg); err != nil {
		return nil, err
	}
	return storage.New(&cfg), nil
}

func newNotification() (*notification.Client, error) {
	var cfg notification.Config
	if err := envconfig.Process("sindico_notification", &cfg); err != nil {
		return nil, err
	}
	return notification.New(&cfg), nil
}

func Run() {
	ctrls, err := newControllers()
	if err != nil {
		log.Error("failed to build controllers", "err", err)
		return
	}
	run(ctrls)
	select {}
}

func newControllers() ([]Controller, error) {
	ctrls := []Controller{}

	ctrl, err := newEtcdBackup()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build etcdbackup ctrl")
	}
	ctrls = append(ctrls, ctrl)

	ctrl, err = newKubeWatch()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build kubewatch ctrl")
	}
	ctrls = append(ctrls, ctrl)

	ctrl, err = newSrebot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build srebot ctrl")
	}
	ctrls = append(ctrls, ctrl)

	ctrl, err = newWatchdog()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build watchdog ctrl")
	}
	ctrls = append(ctrls, ctrl)

	return ctrls, nil
}

func newEtcdBackup() (Controller, error) {
	k, err := newK8s()
	if err != nil {
		return nil, err
	}
	st, err := newStorage()
	if err != nil {
		return nil, err
	}
	nt, err := newNotification()
	if err != nil {
		return nil, err
	}
	ctrl := etcdbackup.NewController(k, st, nt)
	return ctrl, nil
}

func newKubeWatch() (Controller, error) {
	k, err := newK8s()
	if err != nil {
		return nil, err
	}
	nt, err := newNotification()
	if err != nil {
		return nil, err
	}
	ctrl := kubewatch.NewController(k, nt)
	return ctrl, nil
}

func newSrebot() (Controller, error) {
	k, err := newK8s()
	if err != nil {
		return nil, err
	}
	ctrl := srebot.NewController(k)
	return ctrl, nil
}

func newWatchdog() (Controller, error) {
	k, err := newK8s()
	if err != nil {
		return nil, err
	}
	ctrl := watchdog.NewController(k)
	return ctrl, nil
}

func run(ctrls []Controller) {
	ctx := context.Background()
	for _, ctrl := range ctrls {
		go ctrl.Run(ctx.Done())
	}
}
