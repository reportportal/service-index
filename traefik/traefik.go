package traefik

import (
	"errors"
	"github.com/reportportal/service-index/aggregator"
	"gopkg.in/resty.v1"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Providers struct {
	Docker *Provider `json:"docker,omitempty"`
}

type Provider struct {
	Backends map[string]*Backend `json:"backends,omitempty"`
}
type Backend struct {
	Servers map[string]*Server `json:"servers,omitempty"`
}
type Server struct {
	URL    string `json:"url"`
	Weight int    `json:"weight"`
}

type Aggregator struct {
	r     *resty.Client
	lbURL string
}

func NewAggregator(traefikURL string, timeout time.Duration) *Aggregator {
	return &Aggregator{
		r: resty.NewWithClient(&http.Client{
			Timeout: timeout,
		}),
		lbURL: traefikURL,
	}
}

func (a *Aggregator) AggregateHealth() map[string]interface{} {
	return a.aggregate(func(ni *aggregator.NodeInfo) (interface{}, error) {
		var rs map[string]interface{}
		if "" != ni.GetHealthCheckURL() {
			_, e := a.r.R().SetResult(&rs).SetError(&rs).Get(ni.GetHealthCheckURL())
			if nil != e {
				rs = map[string]interface{}{"status": "DOWN"}
			}
		} else {
			rs = map[string]interface{}{"status": "UNKNOWN"}
		}

		return rs, nil
	})
}

func (a *Aggregator) AggregateInfo() map[string]interface{} {
	return a.aggregate(func(info *aggregator.NodeInfo) (interface{}, error) {
		var rs map[string]interface{}
		_, e := a.r.R().SetResult(&rs).Get(info.GetStatusPageURL())
		if nil != e {
			log.Println(e)
			return nil, e
		}
		if nil == rs {
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
