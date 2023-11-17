package k8s

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	// all auth types are supported
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
)

const (
	domainPattern = "%s.svc.%s"
	//nolint:gosec
	nsSecret      = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	labelSelector = "app=reportportal"
)

// Aggregator is an info/health aggregator implementation for k8s
type Aggregator struct {
	localDomain string
	ns          string
	clientset   *kubernetes.Clientset
	r           *resty.Client
}

// NodeInfo embeds node-related information
type NodeInfo struct {
	srv            string
	portName       string
	infoEndpoint   string
	healthEndpoint string
}

// NewAggregator creates new k8s aggregator
func NewAggregator(timeout time.Duration) (*Aggregator, error) {
	ns, err := getCurrentNamespace()
	if err != nil {
		return nil, fmt.Errorf("unable to find out current namespace: %w", err)
	}

	log.Infof("Namespace: %s", ns)
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to get cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("Unable to create k8s client: %v", err)

		return nil, fmt.Errorf("unable to create k8s client: %w", err)
	}

	clusterDomain := getClusterDomain()

	return &Aggregator{
		clientset:   clientset,
		localDomain: fmt.Sprintf(domainPattern, ns, clusterDomain),
		r: resty.NewWithClient(&http.Client{
			Timeout: timeout,
		}).SetScheme("http"),
		ns: ns,
	}, nil
}

// AggregateHealth aggregates health info
func (a *Aggregator) AggregateHealth() map[string]interface{} {
	return a.aggregate(func(ni *NodeInfo) (interface{}, error) {
		var rs map[string]interface{}
		_, e := a.r.R().SetSRV(&resty.SRVRecord{Service: ni.portName, Domain: ni.srv}).SetResult(&rs).SetError(&rs).Get(ni.healthEndpoint)
		if nil != e {
			log.Errorf("Health check error for service [%s] failed: %s", ni.srv, e.Error())
			rs = map[string]interface{}{"status": "DOWN"}
		}

		return rs, nil
	})
}

// AggregateInfo aggregates info
func (a *Aggregator) AggregateInfo() map[string]interface{} {
	return a.aggregate(func(ni *NodeInfo) (interface{}, error) {
		var rs map[string]interface{}
		_, e := a.r.R().SetSRV(&resty.SRVRecord{Service: ni.portName, Domain: ni.srv}).SetResult(&rs).Get(ni.infoEndpoint)
		if nil != e {
			log.Errorf("Unable to aggregate info: %v", e)

			return nil, fmt.Errorf("unable to aggregate info: %w", e)
		}
		if nil == rs {
			log.Error("Unable to collect info endpoint response")

			return nil, errors.New("response is empty")
		}

		return rs, nil
	})
}

func (a *Aggregator) aggregate(f func(ni *NodeInfo) (interface{}, error)) map[string]interface{} {
	log.Debug("Aggregating node information")
	nodesInfo, err := a.getNodesInfo()
	if err != nil {
		log.Errorf("Unable to aggregate node information: %v", err)

		return map[string]interface{}{}
	}

	nodeLen := len(nodesInfo)
	aggregated := make(map[string]interface{}, nodeLen)
	var wg sync.WaitGroup

	wg.Add(nodeLen)
	var mu sync.Mutex
	for node, info := range nodesInfo {
		go func(n string, ni *NodeInfo) {
			defer wg.Done()
			res, err := f(ni)
			if nil == err {
				mu.Lock()
				aggregated[n] = res
				mu.Unlock()
			}
		}(node, info)
	}
	wg.Wait()

	return aggregated
}

func (a *Aggregator) getNodesInfo() (map[string]*NodeInfo, error) {
	services, err := a.clientset.CoreV1().Services(a.ns).List(
		context.Background(),
		metav1.ListOptions{
			LabelSelector: labelSelector,
		})
	if err != nil {
		return nil, fmt.Errorf("unable to aggregate nodes info: %w", err)
	}

	srvCount := len(services.Items)
	log.Infof("Selected [%d] ReportPortal's services", srvCount)
	nodesInfo := make(map[string]*NodeInfo, srvCount)
	for _, srv := range services.Items {
		log.Debugf("Info found for service %s", srv.GetName())

		srvName := srv.GetAnnotations()["service"]
		if srvName == "" {
			continue
		}

		ni := &NodeInfo{srv: srv.GetName() + "." + a.localDomain}
		if ie, ok := srv.GetAnnotations()["infoEndpoint"]; ok {
			ni.infoEndpoint = ie
		} else {
			ni.infoEndpoint = "/info"
		}
		if he, ok := srv.GetAnnotations()["healthEndpoint"]; ok {
			ni.healthEndpoint = he
		} else {
			ni.healthEndpoint = "/health"
		}

		if len(srv.Spec.Ports) > 0 {
			ni.portName = srv.Spec.Ports[0].Name
		}

		nodesInfo[srvName] = ni
	}

	return nodesInfo, nil
}

func getCurrentNamespace() (string, error) {
	ns, err := os.ReadFile(nsSecret)
	if err != nil {
		return "", fmt.Errorf("failed to read namespace secret file: %w", err)
	}

	return string(ns), nil
}

// GetClusterDomain returns Kubernetes cluster domain, default to "cluster.local"
func getClusterDomain() string {
	apiSvc := "kubernetes.default.svc"

	clusterDomain := "cluster.local"

	cname, err := net.LookupCNAME(apiSvc)
	if err != nil {
		return clusterDomain
	}

	clusterDomain = strings.TrimPrefix(cname, apiSvc)
	clusterDomain = strings.Trim(clusterDomain, ".")
	log.Infof("Cluster Domain [%s] Detected", clusterDomain)

	return clusterDomain
}
