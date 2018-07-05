# Sindico

A executable that integrates useful kubernetes 1.9 controllers.

## How it works?
It just runs each controller in a goroutine. A controller must implement the interface:

```go
type Controller interface {
	Run(stopCh <-chan struct{})
}
```

## Global Environment Variables

Used to configure the kubernetes, storage and notification clients. For now only Slack and S3 are supported.

| Env | Description | Default |
|---|---|---|
| SINDICO\_K8S\_CONFIG\_FILE | kubectl config file  | |
| SINDICO\_NOTIFICATION\_AVATAR | notification avatar | |
| SINDICO\_NOTIFICATION\_TOKEN | notification token | |
| SINDICO\_NOTIFICATION\_USERNAME | notification username | sindico |
| SINDICO\_STORAGE\_KEY | storage key | |
| SINDICO\_STORAGE\_SECRET | storage secret | |
| SINDICO\_STORAGE\_REGION | storage region | us-east-1 |
| SINDICO\_STORAGE\_BUCKET | storage bucket | sindico |

## Controllers

### Etcdbackup

Runs `etcdctl backup` via the kubernetes exec api and put the resulting tgz file on
the storage bucket (is there a better way of doing this?).

| Env | Description | Default |
|---|---|---|
| SINDICO\_ETCD\_BACKUP\_INTERVAL | backup interval  | 6h |
| SINDICO\_ETCD\_BACKUP\_DIR | backup directory | etcd-backup |
| SINDICO\_ETCD\_BACKUP\_DISABLED | disable the controller | |
| SINDICO\_ETCD\_BACKUP\_NOTIFICATION\_CHANNEL | notification channel | #alerts |

### Kubewatch

Checks for crashed and not ready pods using the notification client to report the results.

| Env | Description | Default |
|---|---|---|
| SINDICO\_KUBE\_WATCH\_CIRCLE\_TIME | check interval (minutes) | 5 |
| SINDICO\_KUBE\_WATCH\_K8S\_ENV | env description | production |
| SINDICO\_KUBE\_WATCH\_NOT\_READY\_THRESHOLD | % not ready pods | 60 |
| SINDICO\_KUBE\_WATCH\_IGNORE\_NS\_REGEXP | regexp for namespaces to be ignored | default |
| SINDICO\_KUBE\_WATCH\_TEAM\_NS\_ANNOTATION | namespace annotation used to get the notification team | teresa.io/team |
| SINDICO\_KUBE\_WATCH\_NOTIFICATION\_CHANNEL | notification channel | #alerts |

### Srebot

A Slack bot to change deploy replicas.

| Env | Description | Default |
|---|---|---|
| SINDICO\_SRE\_BOT\_SLACK\_TOKEN | slack token | |
| SINDICO\_SRE\_BOT\_ADMINS | comma separated list of admins | admin |
| SINDICO\_SRE\_BOT\_CMD\_PREFIX | cmd prefix | production |

Example usage: `!cmdprefix-set-replicas namespace deployname 0`

### Watchdog

Enforces maximum values for requests and hpa replicas.

| Env | Description | Default |
|---|---|---|
| SINDICO\_WATCHDOG\_LIMITS\_INTERVAL | check interval for limits | 5m |
| SINDICO\_WATCHDOG\_LIMITS\_IGNORE\_NS\_REGEXP | regexp for namespaces to be ignored | nginx-.+\|sindico\|default\|kube-.+ |
| SINDICO\_WATCHDOG\_LIMITS\_REQUEST\_CPU | maximum cpu request | 100m |
| SINDICO\_WATCHDOG\_LIMITS\_REQUEST\_MEMORY | maximum memory request | 512Mi |
| SINDICO\_WATCHDOG\_HPA\_INTERVAL | check interval for hpa | 5m |
| SINDICO\_WATCHDOG\_HPA\_IGNORE\_NS\_REGEXP | regexp for namespaces to be ignored | nginx-.+\|sindico\|default\|kube-.+ |
| SINDICO\_WATCHDOG\_HPA\_MAX\_REPLICAS | maximum hpa max replicas | 2 |
| SINDICO\_WATCHDOG\_HPA\_MIN\_REPLICAS | maximum hpa min replicas | 2 |

## Deploying

Edit the env vars in sindico.yaml and after that:

```
$ kubectl create ns sindico
$ kubectl create -f sindico.yaml -n sindico
```

## TODO

- Tests
- Better controllers, the etcd backup method is specially brittle
