package k8s

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/luizalabs/sindico/controllers/kubewatch"
	"github.com/pkg/errors"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	startHeading = []byte{'\u0001'}
)

type RoundTripCallback func(conn *websocket.Conn, resp *http.Response, err error, stderr io.Writer, stdout io.Writer) error

type WebsocketRoundTripper struct {
	Dialer *websocket.Dialer
	Do     RoundTripCallback
	stderr io.Writer
	stdout io.Writer
}

func (d *WebsocketRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	conn, resp, err := d.Dialer.Dial(r.URL.String(), r.Header)
	if err == nil {
		defer conn.Close()
	}
	return resp, d.Do(conn, resp, err, d.stderr, d.stdout)
}

func WebsocketCallback(ws *websocket.Conn, resp *http.Response, err error, stderr io.Writer, stdout io.Writer) error {
	if err != nil {
		msg := "can't connect to console"
		if resp != nil {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			msg = fmt.Sprintf("%s http status=%d body=%s", msg, resp.StatusCode, buf.String())
		}
		return errors.Wrap(err, msg)
	}
	for {
		_, body, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return nil
			}
			return errors.Wrap(err, "failed to read websocket message")
		}
		tmp := bytes.SplitN(body, startHeading, 2)
		if stderr != nil {
			_, err = stderr.Write(tmp[0])
			if err != nil {
				return errors.Wrap(err, "failed to write to stderr")
			}
		}
		if stdout != nil {
			_, err = stdout.Write(tmp[1])
			if err != nil {
				return errors.Wrap(err, "failed to write to stdout")
			}
		}
	}
}

func (c *Client) FindPods(namespace, labelSelector string) ([]string, error) {
	opts := metav1.ListOptions{LabelSelector: labelSelector}
	pl, err := c.clientset.CoreV1().Pods(namespace).List(opts)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(pl.Items))
	for i, item := range pl.Items {
		names[i] = item.Name
	}
	return names, nil
}

func (c *Client) DeletePod(namespace, pod string) error {
	return c.clientset.CoreV1().Pods(namespace).Delete(pod, &metav1.DeleteOptions{})
}

func (c *Client) SetReplicas(namespace, deploy string, replicas int32) error {
	d, err := c.clientset.AppsV1beta2().Deployments(namespace).Get(deploy, metav1.GetOptions{})
	if err != nil {
		return err
	}
	d.Spec.Replicas = &replicas
	_, err = c.clientset.AppsV1beta2().Deployments(namespace).Update(d)
	return err
}

func (c *Client) execRequest(pod, container, namespace, cmd string) (*http.Request, error) {
	u, err := url.Parse(c.cfg.Host)
	if err != nil {
		return nil, err
	}
	// gorilla/websocket expecst wss:// or ws:// urls
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return nil, fmt.Errorf("malformed url %s", u.String())
	}
	qs := "stdout=true&stderr=true&command="
	qs = fmt.Sprintf("%s%s", qs, strings.Join(strings.Split(cmd, " "), "&command="))
	if container != "" {
		qs = fmt.Sprintf("%s&container=%s", qs, container)
	}
	u.Path = fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/exec", namespace, pod)
	u.RawQuery = qs
	req := &http.Request{Method: http.MethodGet, URL: u}
	return req, nil
}

func (c *Client) roundTripper(stderr io.Writer, stdout io.Writer) (http.RoundTripper, error) {
	tlsConfig, err := rest.TLSConfigFor(c.cfg)
	if err != nil {
		return nil, err
	}
	dialer := &websocket.Dialer{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: tlsConfig,
	}
	rt := &WebsocketRoundTripper{
		Do:     WebsocketCallback,
		Dialer: dialer,
		stderr: stderr,
		stdout: stdout,
	}
	return rest.HTTPWrappersForConfig(c.cfg, rt)
}

func (c *Client) Exec(pod, container, namespace, cmd string, stderr io.Writer, stdout io.Writer) (*http.Response, error) {
	wrappedRoundTripper, err := c.roundTripper(stderr, stdout)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build roundtripper")
	}
	req, err := c.execRequest(pod, container, namespace, cmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build exec request")
	}
	return wrappedRoundTripper.RoundTrip(req)
}

func (c *Client) NewClientset() (kubernetes.Interface, error) {
	return kubernetes.NewForConfig(c.cfg)
}

func (c *Client) List(namespace string) ([]kubewatch.Pod, error) {
	podList, err := c.clientset.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return convertPodList(podList.Items), nil
}

func (c *Client) GetLabelValue(namespace, label string) (string, error) {
	ns, err := c.clientset.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	if err != nil {
		return "", nil
	}
	return ns.Labels[label], nil
}

func convertPodList(items []k8sv1.Pod) []kubewatch.Pod {
	pods := make([]kubewatch.Pod, 0)
	for _, pod := range items {
		for _, status := range pod.Status.ContainerStatuses {
			state := "Running"
			if status.State.Waiting != nil {
				state = status.State.Waiting.Reason
			} else if status.State.Terminated != nil {
				state = status.State.Terminated.Reason
			}
			pods = append(pods, kubewatch.Pod{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Status:    state,
				Ready:     status.Ready})
		}
	}
	return pods
}

func newInClusterK8sClient() (*Client, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build incluster cfg")
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build clientset")
	}
	return &Client{clientset: clientset, cfg: cfg}, nil
}

func newOutOfClusterK8sClient(conf *Config) (*Client, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", conf.ConfigFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build outofcluster cfg")
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build clientset")
	}
	return &Client{clientset: clientset, cfg: cfg}, nil
}
