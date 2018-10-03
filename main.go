package main

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/reportportal/service-index/traefik"
	log "github.com/sirupsen/logrus"
	"gopkg.in/reportportal/commons-go.v5/commons"
	"gopkg.in/reportportal/commons-go.v5/conf"
	"gopkg.in/reportportal/commons-go.v5/server"
	"log"
	"net/http"
	"time"
)

const httpClientTimeout = 5 * time.Second

func main() {

	cfg := conf.EmptyConfig()

	rpCfg := struct {
		*conf.ServerConfig
		LbURL string `env:"LB_URL" envDefault:"http://dev.epm-rpp.projects.epam.com:9091/api/providers/docker"`
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

	aggregator := traefik.NewAggregator(rpCfg.LbURL, httpClientTimeout)

	srv.WithRouter(func(router *chi.Mux) {
		router.Use(middleware.Logger)
		router.NotFound(func(w http.ResponseWriter, rq *http.Request) {
			http.Redirect(w, rq, "/ui/404.html", http.StatusFound)
		})

		router.HandleFunc("/composite/info", func(w http.ResponseWriter, r *http.Request) {
			if err := server.WriteJSON(http.StatusOK, aggregator.AggregateInfo(), w); nil != err {
				log.Error(err)
			}
		})
		router.HandleFunc("/composite/health", func(w http.ResponseWriter, r *http.Request) {
			if err := server.WriteJSON(http.StatusOK, aggregator.AggregateHealth(), w); nil != err {
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
	srv.StartServer()
}
