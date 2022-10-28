package main

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/reportportal/commons-go/v5/commons"
	"github.com/reportportal/commons-go/v5/conf"
	"github.com/reportportal/commons-go/v5/server"
	log "github.com/sirupsen/logrus"

	"github.com/reportportal/service-index/aggregator"
	"github.com/reportportal/service-index/k8s"
	"github.com/reportportal/service-index/traefik"
)

const httpClientTimeout = 5 * time.Second

func main() {
	cfg := conf.EmptyConfig()

	rpCfg := struct {
		*conf.ServerConfig
		K8sMode       bool   `env:"K8S_MODE" envDefault:"false"`
		TraefikV2Mode bool   `env:"TRAEFIK_V2_MODE" envDefault:"false"`
		TraefikLbURL  string `env:"LB_URL" envDefault:"http://localhost:9091"`
		LogLevel      string `env:"LOG_LEVEL" envDefault:"info"`
		Path          string `env:"RESOURCE_PATH" envDefault:""`
	}{
		ServerConfig: cfg,
	}

	err := conf.LoadConfig(&rpCfg)
	if nil != err {
		log.Fatalf("Cannot load config %v", err)
	}
	ll, err := log.ParseLevel(rpCfg.LogLevel)
	if err != nil {
		log.Fatalf("Incorrect log level provided: %v", err)
	}
	log.SetLevel(ll)

	info := commons.GetBuildInfo()
	info.Name = "Index Service"

	srv := server.New(rpCfg.ServerConfig, info)

	log.Infof("K8S mode enabled: %t", rpCfg.K8sMode)
	var aggreg aggregator.Aggregator
	if rpCfg.K8sMode {
		aggreg, err = k8s.NewAggregator(httpClientTimeout)
		if nil != err {
			log.Fatalf("Incorrect K8S config %s", err.Error())
		}
	} else {
		aggreg = traefik.NewAggregator(rpCfg.TraefikLbURL, rpCfg.TraefikV2Mode, httpClientTimeout)
	}

	srv.WithRouter(func(router *chi.Mux) {
		router.Use(middleware.Logger)
		router.NotFound(func(w http.ResponseWriter, rq *http.Request) {
			http.Redirect(w, rq, rpCfg.Path+"/ui/#notfound", http.StatusFound)
		})

		router.HandleFunc(rpCfg.Path+"/composite/info", func(w http.ResponseWriter, r *http.Request) {
			if err := server.WriteJSON(http.StatusOK, aggreg.AggregateInfo(), w); nil != err {
				log.Error(err)
			}
		})
		router.HandleFunc(rpCfg.Path+"/composite/health", func(w http.ResponseWriter, r *http.Request) {
			if err := server.WriteJSON(http.StatusOK, aggreg.AggregateHealth(), w); nil != err {
				log.Error(err)
			}
		})
		router.HandleFunc(rpCfg.Path, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, rpCfg.Path+"/ui/", http.StatusFound)
		})
		router.HandleFunc(rpCfg.Path+"/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, rpCfg.Path+"/ui/", http.StatusFound)
		})
		router.HandleFunc(rpCfg.Path+"/ui", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, rpCfg.Path+"/ui/", http.StatusFound)
		})
	})
	srv.StartServer()
}
