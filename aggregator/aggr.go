package aggregator

import "net/url"

type (
	Aggregator interface {
		AggregateInfo() map[string]interface{}
		AggregateHealth() map[string]interface{}
	}

	NodeInfo struct {
		URL string
	}
)

func (ni *NodeInfo) GetStatusPageURL() string {
	return ni.URL + "/info"
}
func (ni *NodeInfo) GetHealthCheckURL() string {
	return ni.URL + "/health"
}

func (ni *NodeInfo) BuildURL(h, path string) string {
	u, _ := url.Parse(h)
	//u.Host = h
	u.Path = path
	return u.String()
}
