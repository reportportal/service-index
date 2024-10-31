package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/reportportal/service-index/buildinfo"
)

type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func ExampleRpServer() {
	rpConf := EmptyConfig()
	_ = LoadConfig(rpConf)
	rp := New(rpConf, buildinfo.GetBuildInfo())

	rp.WithRouter(func(router *chi.Mux) {
		router.Get("/ping", func(w http.ResponseWriter, rq *http.Request) {
			if err := WriteJSON(http.StatusOK, Person{"av", 20}, w); err != nil {
				logrus.Error(err)
			}
		})
	})

	rp.StartServer()
}
