package aggregator

import (
	log "github.com/sirupsen/logrus"
	"net/url"
)

type (
	//Aggregator collects information from all available services
	Aggregator interface {
		//AggregateInfo collects information from info endpoints
		AggregateInfo() map[string]interface{}

		//AggregateHealth aggregates information from health endpoints
		AggregateHealth() map[string]interface{}
	}

	//NodeInfo embeds node-related information
	NodeInfo struct {
		URL string
	}
)

//GetInfoEndpoint returns info endpoint URL
func (ni *NodeInfo) GetInfoEndpoint() string {
	return ni.URL + "/info"
}

//GetHealthEndpoint returns health check URL
func (ni *NodeInfo) GetHealthEndpoint() string {
	return ni.URL + "/health"
}

func (ni *NodeInfo) buildURL(h, path string) string {
	u, err := url.Parse(h)
	if nil != err {
		log.Error(err)
		return ""
	}
	//u.Host = h
	u.Path = path
	return u.String()
}
