package main

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/reportportal/commons-go/commons"
	"github.com/reportportal/commons-go/conf"
	"github.com/reportportal/commons-go/server"
	"github.com/reportportal/service-index/aggregator"
	"github.com/reportportal/service-index/k8s"
	"github.com/reportportal/service-index/traefik"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

const httpClientTimeout = 5 * time.Second

func main() {

	cfg := conf.EmptyConfig()

	rpCfg := struct {
		*conf.ServerConfig
		K8sMode bool   `env:"K8S_MODE" envDefault:"false"`
		K8SNs   string `env:"K8S_NAMESPACE" envDefault:"default"`

		TraefikLbURL string `env:"LB_URL" envDefault:"http://localhost:9091"`
	}{
		ServerConfig: cfg,
	}

	err := conf.LoadConfig(&rpCfg)
	if nil != err {
		log.Fatalf("Cannot load config %s", err.Error())
	}

	info := commons.GetBuildInfo()
	info.Name = "Index Service"

	srv := server.New(rpCfg.ServerConfig, info)

	var aggreg aggregator.Aggregator
	if rpCfg.K8sMode {
		aggreg, err = k8s.NewAggregator(rpCfg.K8SNs, httpClientTimeout)
		if nil != err {
			log.Fatalf("Incorrect K8S config %s", err.Error())
		}
	} else {
		aggreg = traefik.NewAggregator(rpCfg.TraefikLbURL, httpClientTimeout)
	}

	srv.WithRouter(func(router *chi.Mux) {
		router.Use(middleware.Logger)
		router.NotFound(func(w http.ResponseWriter, rq *http.Request) {
			http.Redirect(w, rq, "/ui/#notfound", http.StatusFound)
		})

		router.HandleFunc("/composite/info", func(w http.ResponseWriter, r *http.Request) {
			if err := server.WriteJSON(http.StatusOK, aggreg.AggregateInfo(), w); nil != err {
				log.Error(err)
			}
		})
		router.HandleFunc("/composite/health", func(w http.ResponseWriter, r *http.Request) {
			if err := server.WriteJSON(http.StatusOK, aggreg.AggregateHealth(), w); nil != err {
				log.Error(err)
			}
		})
		router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/ui/", http.StatusFound)
		})
		router.HandleFunc("/ui", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/ui/", http.StatusFound)
		})

	})
	fmt.Println(info)
	srv.StartServer()
}
