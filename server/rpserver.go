package server

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	log "github.com/sirupsen/logrus"

	"github.com/reportportal/service-index/buildinfo"
)

// RpServer represents ReportPortal microservice instance
type RpServer struct {
	mux       *chi.Mux
	cfg       *Config
	buildInfo *buildinfo.BuildInfo
	hChecks   []HealthCheck
}

// New creates new instance of RpServer struct
func New(cfg *Config, buildInfo *buildinfo.BuildInfo) *RpServer {
	srv := &RpServer{
		mux:       chi.NewMux(),
		cfg:       cfg,
		hChecks:   []HealthCheck{},
		buildInfo: buildInfo,
	}

	srv.mux.Use(middleware.Recoverer)

	return srv
}

// WithRouter gives access to Chi router to add route and perform other modifications
func (srv *RpServer) WithRouter(f func(router *chi.Mux)) {
	f(srv.mux)
}

// AddHandler is preferred way to add handler to the server.
// Allows to return error which is representation of HTTP Response
func (srv *RpServer) AddHandler(method, pattern string, f func(w http.ResponseWriter, r *http.Request) error) {
	srv.mux.Method(method, pattern, Handler{f})
}

// AddHealthCheck adds health check
func (srv *RpServer) AddHealthCheck(h HealthCheck) {
	srv.hChecks = append(srv.hChecks, h)
}

// AddHealthCheckFunc adds health check function
func (srv *RpServer) AddHealthCheckFunc(f func() error) {
	srv.hChecks = append(srv.hChecks, HealthCheckFunc(f))
}

// StartServer starts HTTP server
func (srv *RpServer) StartServer() {
	srv.initDefaultRoutes()

	// listen and server on mentioned port
	log.Printf("Starting on port %d", srv.cfg.Port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(srv.cfg.Port), srv.mux))
}

// initDefaultRoutes initializes default routes

func (srv *RpServer) initDefaultRoutes() {
	srv.mux.Get("/health", srv.healthHandler)
	srv.mux.Get("/info", srv.infoHandler)
}

func (srv *RpServer) infoHandler(w http.ResponseWriter, rq *http.Request) {
	bi := map[string]interface{}{"build": srv.buildInfo}
	if err := WriteJSON(http.StatusOK, bi, w); err != nil {
		log.Error(err)
	}
}

func (srv *RpServer) healthHandler(w http.ResponseWriter, rq *http.Request) {
	// collect status results
	var errs []string
	for _, hc := range srv.hChecks {
		if err := hc.Check(); nil != err {
			errs = append(errs, err.Error())
		}
	}

	rs := map[string]interface{}{}
	status := http.StatusOK
	if len(errs) > 0 {
		rs["status"] = "DOWN"
		rs["errors"] = errs
		status = http.StatusBadRequest
	} else {
		rs["status"] = "UP"
	}

	if err := WriteJSON(status, rs, w); err != nil {
		log.Error(err)
	}
}
