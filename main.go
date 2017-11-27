package main

import (
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
	var aggregated = make(map[string]interface{}, len(nodesInfo))
	for node, info := range nodesInfo {
		var rs map[string]interface{}

		if "" != info.getHealthCheckURL() {
			_, e := sling.New().Client(a.c).Base(info.BaseURL).Get(info.getHealthCheckURL()).Receive(&rs, &rs)
			if nil != e {
				rs = make(map[string]interface{}, 1)
				rs["status"] = "DOWN"
			}
		} else {
			rs = make(map[string]interface{}, 1)
			rs["status"] = "UNKNOWN"
		}

		aggregated[node] = rs
	}
	return aggregated
}

func (a *compositeAggregator) aggregateInfo(nodesInfo map[string]*nodeInfo) map[string]interface{} {
	var aggregated = make(map[string]interface{}, len(nodesInfo))
	for node, info := range nodesInfo {
		var rs map[string]interface{}
		_, e := sling.New().Client(a.c).Base(info.BaseURL).Get(info.getStatusPageURL()).ReceiveSuccess(&rs)
		if nil != e {
			log.Println(e)
			continue
		}
		if nil != rs {
			aggregated[node] = rs
		}

	}
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
			for _, inst := range instances {
				tagsMap := map[string]string{}
				parseKVTag(inst.Service.Tags, tagsMap)

				var ni nodeInfo
				ni.BaseURL = fmt.Sprintf("http://%s:%d/", inst.Service.Address, inst.Service.Port)
				ni.Tags = tagsMap
				nodesInfo[strings.ToUpper(k)] = &ni
			}

		}

		return nodesInfo, nil
	})
	return nodesInfo.(map[string]*nodeInfo)
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
