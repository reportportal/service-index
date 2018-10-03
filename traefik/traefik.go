package traefik

import (
	"errors"
	"github.com/reportportal/service-index/aggregator"
	log "github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
	"net/http"
	"strings"
	"sync"
	"time"
)

//Providers represents traefik response model
type Providers struct {
	Docker *Provider `json:"docker,omitempty"`
}

//Provider represents traefik response model
type Provider struct {
	Backends map[string]*Backend `json:"backends,omitempty"`
}

//Backend represents traefik response model
type Backend struct {
	Servers map[string]*Server `json:"servers,omitempty"`
}

//Server represents traefik response model
type Server struct {
	URL    string `json:"url"`
	Weight int    `json:"weight"`
}

//Aggregator represents traefik response model
type Aggregator struct {
	r     *resty.Client
	lbURL string
}

//NewAggregator creates new traefik aggregator
func NewAggregator(traefikURL string, timeout time.Duration) *Aggregator {
	return &Aggregator{
		r: resty.NewWithClient(&http.Client{
			Timeout: timeout,
		}),
		lbURL: traefikURL,
	}
}

//AggregateHealth aggregates health info
func (a *Aggregator) AggregateHealth() map[string]interface{} {
	return a.aggregate(func(ni *aggregator.NodeInfo) (interface{}, error) {
		var rs map[string]interface{}
		if "" != ni.GetHealthEndpoint() {
			_, e := a.r.R().SetResult(&rs).SetError(&rs).Get(ni.GetHealthEndpoint())
			if nil != e {
				rs = map[string]interface{}{"status": "DOWN"}
			}
		} else {
			rs = map[string]interface{}{"status": "UNKNOWN"}
		}

		return rs, nil
	})
}

//AggregateInfo aggregates info
func (a *Aggregator) AggregateInfo() map[string]interface{} {
	return a.aggregate(func(info *aggregator.NodeInfo) (interface{}, error) {
		var rs map[string]interface{}
		_, e := a.r.R().SetResult(&rs).Get(info.GetInfoEndpoint())
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

func (a *Aggregator) aggregate(f func(ni *aggregator.NodeInfo) (interface{}, error)) map[string]interface{} {

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
		go func(node string, info *aggregator.NodeInfo) {
			defer wg.Done()
			res, err := f(info)
			if nil == err {
				mu.Lock()
				aggregated[node] = res
				mu.Unlock()
			}
		}(node, info)
	}
	wg.Wait()
	return aggregated
}

func (a *Aggregator) getNodesInfo() (map[string]*aggregator.NodeInfo, error) {

	var provider Provider
	_, err := a.r.R().SetResult(&provider).Get(a.lbURL)
	if nil != err {
		return nil, err
	}

	nodesInfo := make(map[string]*aggregator.NodeInfo, len(provider.Backends))

	if nil == err {
		for bName, b := range provider.Backends {
			backName := bName[strings.LastIndex(bName, "backend-")+len("backend-"):]
			nodesInfo[backName] = &aggregator.NodeInfo{URL: getFirstNode(b.Servers).URL}
		}
	}

	return nodesInfo, nil
}

func getFirstNode(m map[string]*Server) *Server {
	for _, v := range m {
		return v
	}
	return nil
}
