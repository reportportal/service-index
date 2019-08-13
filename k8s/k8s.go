package k8s

import (
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	//all auth types are supported
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"net/http"
	"sync"
	"time"
)

const domainPattern = "%s.svc.cluster.local"

//Aggregator is an info/health aggregator implementation for k8s
type Aggregator struct {
	localDomain string
	clientset   *kubernetes.Clientset
	r           *resty.Client
}

//NodeInfo embeds node-related information
type NodeInfo struct {
	srv            string
	infoEndpoint   string
	healthEndpoint string
}

//NewAggregator creates new k8s aggregator
func NewAggregator(ns string, timeout time.Duration) (*Aggregator, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &Aggregator{
		clientset:   clientset,
		localDomain: fmt.Sprintf(domainPattern, ns),
		r: resty.NewWithClient(&http.Client{
			Timeout: timeout,
		}),
	}, nil
}

//AggregateHealth aggregates health info
func (a *Aggregator) AggregateHealth() map[string]interface{} {
	return a.aggregate(func(ni *NodeInfo) (interface{}, error) {
		var rs map[string]interface{}
		_, e := a.r.R().SetSRV(&resty.SRVRecord{Service: ni.srv, Domain: a.localDomain}).SetResult(&rs).SetError(&rs).Get(ni.healthEndpoint)
		if nil != e {
			rs = map[string]interface{}{"status": "DOWN"}
		}

		return rs, nil
	})
}

//AggregateInfo aggregates info
func (a *Aggregator) AggregateInfo() map[string]interface{} {
	return a.aggregate(func(ni *NodeInfo) (interface{}, error) {
		var rs map[string]interface{}
		_, e := a.r.R().SetSRV(&resty.SRVRecord{Service: ni.srv, Domain: a.localDomain}).SetResult(&rs).Get(ni.infoEndpoint)
		if nil != e {
			log.Errorf("Unable to aggregate info: %v", e)
			return nil, e
		}
		if nil == rs {
			log.Error("Unable to collect info endpoint response")
			return nil, errors.New("response is empty")
		}
		return rs, nil
	})
}

func (a *Aggregator) aggregate(f func(ni *NodeInfo) (interface{}, error)) map[string]interface{} {

	nodesInfo, err := a.getNodesInfo()
	if err != nil {
		return map[string]interface{}{}
	}

	nodeLen := len(nodesInfo)
	var aggregated = make(map[string]interface{}, nodeLen)
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

	services, err := a.clientset.CoreV1().Services("reportportal").List(metav1.ListOptions{
		LabelSelector: "app=reportportal",
	})
	if err != nil {
		return nil, err
	}

	nodesInfo := make(map[string]*NodeInfo, len(services.Items))
	for _, srv := range services.Items {
		log.Debugf("Info found for service %s", srv.GetName())

		var srvName = srv.GetAnnotations()["service"]
		ni := &NodeInfo{srv: srv.GetName()}
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

		nodesInfo[srvName] = ni
	}
	return nodesInfo, nil
}
