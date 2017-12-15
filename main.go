package main

import (
	"errors"
	"fmt"
	"github.com/dghubble/sling"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/hashicorp/consul/api"
	"gopkg.in/reportportal/commons-go.v1/commons"
	"gopkg.in/reportportal/commons-go.v1/conf"
	"gopkg.in/reportportal/commons-go.v1/registry"
	"gopkg.in/reportportal/commons-go.v1/server"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

func main() {

	cfg := conf.EmptyConfig()

	cfg.Consul.Address = "registry:8500"
	cfg.Consul.Tags = []string{
		"urlprefix-/",
		"traefik.frontend.rule=PathPrefix:/",
		"traefik.backend=index",
	}

	rpCfg := struct {
		ProxyConsul bool `env:"RP_PROXY_CONSUL" envDefault:"false"`
		*conf.RpConfig
	}{
		ProxyConsul: false,
		RpConfig:    cfg,
	}

	err := conf.LoadConfig(&rpCfg)
	if nil != err {
		log.Fatalf("Cannot load config %s", err.Error())
	}
	rpCfg.AppName = "index"

	info := commons.GetBuildInfo()
	info.Name = "Service Index"

	srv := server.New(rpCfg.RpConfig, info)

	aggregator := &compositeAggregator{
		c: &http.Client{
			Timeout: 3 * time.Second,
		},
	}

	srv.WithRouter(func(router *chi.Mux) {
		router.Use(middleware.Logger)
		router.NotFound(func(w http.ResponseWriter, rq *http.Request) {
			http.Redirect(w, rq, "/ui/404.html", http.StatusFound)
		})

		router.HandleFunc("/composite/info", func(w http.ResponseWriter, r *http.Request) {
			server.WriteJSON(http.StatusOK, aggregator.aggregateInfo(getNodesInfo(srv.Sd, true)), w)
		})
		router.HandleFunc("/composite/health", func(w http.ResponseWriter, r *http.Request) {
			server.WriteJSON(http.StatusOK, aggregator.aggregateHealth(getNodesInfo(srv.Sd, false)), w)
		})
		router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/ui/", http.StatusFound)
		})
		router.HandleFunc("/ui", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/ui/", http.StatusFound)
		})

		if true == rpCfg.ProxyConsul {
			u, e := url.Parse("http://" + rpCfg.Consul.Address)
			if e != nil {
				log.Fatal("Cannot parse consul URL")
			}

			proxy := httputil.NewSingleHostReverseProxy(u)
			router.Handle("/consul/*", http.StripPrefix("/consul/", proxy))
			router.Handle("/v1/*", proxy)
		}

	})
	srv.StartServer()
}

func parseKVTag(tags []string, tagsMap map[string]string) {
	for _, tag := range tags {
		kv := strings.Split(tag, "=")
		if 2 == len(kv) {
			tagsMap[kv[0]] = kv[1]
		}
	}
}

func (a *compositeAggregator) aggregateHealth(nodesInfo map[string]*nodeInfo) map[string]interface{} {
	return a.aggregate(nodesInfo, func(ni *nodeInfo) (interface{}, error) {
		var rs map[string]interface{}
		if "" != ni.getHealthCheckURL() {
			_, e := sling.New().Client(a.c).Base(ni.BaseURL).Get(ni.getHealthCheckURL()).Receive(&rs, &rs)
			if nil != e {
				rs = make(map[string]interface{}, 1)
				rs["status"] = "DOWN"
			}
		} else {
			rs = make(map[string]interface{}, 1)
			rs["status"] = "UNKNOWN"
		}

		return rs, nil
	})
}

func (a *compositeAggregator) aggregateInfo(nodesInfo map[string]*nodeInfo) map[string]interface{} {
	return a.aggregate(nodesInfo, func(info *nodeInfo) (interface{}, error) {
		var rs map[string]interface{}
		_, e := sling.New().Client(a.c).Base(info.BaseURL).Get(info.getStatusPageURL()).ReceiveSuccess(&rs)
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

func (a *compositeAggregator) aggregate(nodesInfo map[string]*nodeInfo, f func(ni *nodeInfo) (interface{}, error)) map[string]interface{} {

	nodeLen := len(nodesInfo)
	var aggregated = make(map[string]interface{}, nodeLen)
	var wg sync.WaitGroup

	wg.Add(nodeLen)
	var mu sync.Mutex
	for node, info := range nodesInfo {
		go func(node string, info *nodeInfo) {
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

func getNodesInfo(discovery registry.ServiceDiscovery, passing bool) map[string]*nodeInfo {
	nodesInfo, _ := discovery.DoWithClient(func(client interface{}) (interface{}, error) {
		services, _, e := client.(*api.Client).Catalog().Services(&api.QueryOptions{})
		if nil != e {
			return nil, e
		}
		nodesInfo := make(map[string]*nodeInfo, len(services))
		for k := range services {
			instances, _, e := client.(*api.Client).Health().Service(k, "", passing, &api.QueryOptions{})
			if nil != e {
				return nil, e
			}
			//return node info of first instance
			if len(instances) > 0 {
				inst := findFirstValidInstance(instances)
				if nil != inst {
					tagsMap := map[string]string{}
					parseKVTag(inst.Service.Tags, tagsMap)

					var ni nodeInfo
					ni.BaseURL = fmt.Sprintf("http://%s:%d/", inst.Service.Address, inst.Service.Port)
					ni.Tags = tagsMap
					nodesInfo[strings.ToUpper(k)] = &ni
				}
			}
		}

		return nodesInfo, nil
	})
	return nodesInfo.(map[string]*nodeInfo)
}

func findFirstValidInstance(instances []*api.ServiceEntry) *api.ServiceEntry {
	for _, inst := range instances {
		if "" != inst.Service.Address {
			return inst
		}
	}
	return nil
}

type compositeAggregator struct {
	c *http.Client
}

type nodeInfo struct {
	BaseURL string
	Tags    map[string]string
}

func (ni *nodeInfo) getStatusPageURL() string {
	return ni.Tags["statusPageUrlPath"]
}
func (ni *nodeInfo) getHealthCheckURL() string {
	return ni.Tags["healthCheckUrlPath"]
}
