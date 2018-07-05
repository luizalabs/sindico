package etcdbackup

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	log "github.com/inconshreveable/log15"
	"github.com/kelseyhightower/envconfig"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	kubeNamespace = "kube-system"
	backupCmd     = "etcdctl backup --data-dir /var/etcd/data --backup-dir /tmp/etcd-backup"
	fetchCmd      = "tar czf - -C / tmp/etcd-backup"
	cleanupCmd    = "rm -rf /tmp/etcd-backup"
)

type K8s interface {
	FindPods(namespace, labelSelector string) ([]string, error)
	Exec(pod, container, namespace, cmd string, stderr io.Writer, stdout io.Writer) (*http.Response, error)
}

type Notification interface {
	PostMessage(msg, channel string) error
}

type Storage interface {
	UploadFile(path string, r io.ReadSeeker) error
}

type EtcdBackupConfig struct {
	Interval            time.Duration `split_words:"true" default:"6h"`
	Dir                 string        `split_words:"true" default:"etcd-backup"`
	Disabled            string        `split_words:"true" default:""`
	NotificationChannel string        `split_words:"true" default:"#alerts"`
}

type Controller struct {
	k8s    K8s
	st     Storage
	nt     Notification
	logger log.Logger
}

func NewController(k8s K8s, st Storage, nt Notification) *Controller {
	logger := log.New("controller", "etcdbackup")
	return &Controller{k8s: k8s, st: st, nt: nt, logger: logger}
}

func backupName(dir string) string {
	format := "2006-01-02_15:04:05-07:00"
	return fmt.Sprintf("%s/etcd-backup-%s.tgz", dir, time.Now().Format(format))
}

func (c *Controller) notifyError(msg, channel string, val ...interface{}) {
	c.logger.Error(msg, val...)
	tmp := fmt.Sprintf("*sindico etcdbackup error*: %v", val)
	if err := c.nt.PostMessage(tmp, channel); err != nil {
		c.logger.Error("can't send message", val...)
	}
}

func (c *Controller) Run(stopCh <-chan struct{}) {
	var cfg EtcdBackupConfig
	if err := envconfig.Process("sindico_etcd_backup", &cfg); err != nil {
		c.logger.Error("failed to process env vars", "err", err)
		return
	}
	if cfg.Disabled != "" {
		c.logger.Debug("disabled")
		return
	}
	c.logger.Debug("starting")
	fn := func() { c.backup(&cfg) }
	time.Sleep(10 * time.Minute)
	go wait.JitterUntil(fn, cfg.Interval, 0.1, true, stopCh)
	<-stopCh
	c.logger.Debug("stopped")
}

func (c *Controller) cleanup(cfg *EtcdBackupConfig, pod string) {
	var stderr bytes.Buffer
	_, err := c.k8s.Exec(pod, "", kubeNamespace, cleanupCmd, &stderr, nil)
	resp := stderr.String()
	if err != nil || resp != "" {
		c.notifyError(
			"dir cleaning failed",
			cfg.NotificationChannel,
			"err", err,
			"pod", pod,
			"stderr", resp,
		)
	}
}

func (c *Controller) backup(cfg *EtcdBackupConfig) {
	pods, err := c.k8s.FindPods(kubeNamespace, "k8s-app=etcd-server")
	if err != nil {
		c.notifyError("find pods failed", cfg.NotificationChannel, "err", err)
		return
	}
	n := rand.Intn(len(pods))
	pod := string(pods[n])
	var stderr, stdout bytes.Buffer
	_, err = c.k8s.Exec(pod, "", kubeNamespace, backupCmd, &stderr, nil)
	resp := stderr.String()
	if err != nil || resp != "" {
		c.notifyError(
			"backup failed",
			cfg.NotificationChannel,
			"err", err,
			"pod", pod,
			"stderr", resp,
		)
		return
	}
	defer c.cleanup(cfg, pod)
	stderr.Reset()
	_, err = c.k8s.Exec(pod, "", kubeNamespace, fetchCmd, &stderr, &stdout)
	resp = stderr.String()
	if err != nil || resp != "" {
		c.notifyError(
			"tar creation failed",
			cfg.NotificationChannel,
			"err", err,
			"pod", pod,
			"stderr", resp,
		)
		return
	}
	r := bytes.NewReader(stdout.Bytes())
	fname := backupName(cfg.Dir)
	if err := c.st.UploadFile(fname, r); err != nil {
		c.notifyError(
			"upload failed",
			cfg.NotificationChannel,
			"err", err,
			"pod", pod,
		)
		return
	}
	c.logger.Debug("done", "fname", fname)
}
