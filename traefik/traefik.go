package traefik

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

const (
	traefikV1ProvidersURL = "/api/providers/docker"
	traefikV2ServicesURL  = "/api/http/services"
)

var (
	errEmptyResponse = errors.New("response is empty")
	errGetHealthErr  = errors.New("unable to update health info")
)

// Providers represents traefik response model
type Providers struct {
	Docker *Provider `json:"docker,omitempty"`
}

// Provider represents traefik response model
type Provider struct {
	Backends map[string]*Backend `json:"backends,omitempty"`
}

// Backend represents traefik response model
type Backend struct {
	Servers map[string]*Server `json:"servers,omitempty"`
}

// Server represents traefik response model
type Server struct {
	URL    string `json:"url"`
	Weight int    `json:"weight"`
}

// Aggregator represents traefik response model
type Aggregator struct {
	r          *resty.Client
	traefikURL string
	v2         bool
}

// NodeInfo embeds node-related information
type NodeInfo struct {
	URL string
}

// GetInfoEndpoint returns info endpoint URL
func (ni *NodeInfo) GetInfoEndpoint() string {
	return ni.URL + "/info"
}

// GetHealthEndpoint returns health check URL
func (ni *NodeInfo) GetHealthEndpoint() string {
	return ni.URL + "/health"
}

//nolint:unused
func (ni *NodeInfo) buildURL(h, path string) string {
	u, err := url.Parse(h)
	if nil != err {
		log.Error(err)

		return ""
	}
	// u.Host = h
	u.Path = path

	return u.String()
}

// NewAggregator creates new traefik aggregator
func NewAggregator(traefikURL string, traefikV2 bool, timeout time.Duration) *Aggregator {
	return &Aggregator{
		r: resty.NewWithClient(&http.Client{
			Timeout: timeout,
		}),
		traefikURL: traefikURL,
		v2:         traefikV2,
	}
}

// AggregateHealth aggregates health info
func (a *Aggregator) AggregateHealth() map[string]interface{} {
	return a.aggregate(func(ni *NodeInfo) (interface{}, error) {
		var rs map[string]interface{}
		if ni.GetHealthEndpoint() != "" {
			_, e := a.r.R().SetResult(&rs).SetError(&rs).Get(ni.GetHealthEndpoint())
			if nil != e {
				rs = map[string]interface{}{"status": "DOWN"}
			}
		} else {
			rs = map[string]interface{}{"status": "UNKNOWN"}
		}

		return rs, nil
	})
}

// AggregateInfo aggregates info
func (a *Aggregator) AggregateInfo() map[string]interface{} {
	return a.aggregate(func(info *NodeInfo) (interface{}, error) {
		var rs map[string]interface{}
		_, e := a.r.R().SetResult(&rs).Get(info.GetInfoEndpoint())
		if nil != e {
			log.Errorf("Unable to aggregate info: %v", e)

			return nil, fmt.Errorf("unable to aggregate nodes info: %w", e)
		}
		if nil == rs {
			log.Error("Unable to collect info endpoint response")

			return nil, errEmptyResponse
		}

		return rs, nil
	})
}

func (a *Aggregator) aggregate(f func(ni *NodeInfo) (interface{}, error)) map[string]interface{} {
	var nodesInfo map[string]*NodeInfo
	var err error
	if a.v2 {
		nodesInfo, err = a.getNodesInfoV2()
	} else {
		nodesInfo, err = a.getNodesInfo()
	}

	if err != nil {
		return map[string]interface{}{}
	}

	nodeLen := len(nodesInfo)
	aggregated := make(map[string]interface{}, nodeLen)
	var wg sync.WaitGroup

	wg.Add(nodeLen)
	var mu sync.Mutex
	for node, info := range nodesInfo {
		go func(node string, info *NodeInfo) {
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

func (a *Aggregator) getNodesInfo() (map[string]*NodeInfo, error) {
	var provider Provider
	_, err := a.r.R().SetResult(&provider).Get(a.traefikURL + traefikV1ProvidersURL)
	if nil != err {
		return nil, fmt.Errorf("unable to GET Traefik providers: %w", err)
	}

	nodesInfo := make(map[string]*NodeInfo, len(provider.Backends))

	for bName, b := range provider.Backends {
		backName := bName[strings.LastIndex(bName, "backend-")+len("backend-"):]
		nodesInfo[backName] = &NodeInfo{URL: getFirstNode(b.Servers).URL}
	}

	return nodesInfo, nil
}

func (a *Aggregator) getNodesInfoV2() (map[string]*NodeInfo, error) {
	var serviceInfo []*serviceRepresentation
	rs, err := a.r.R().SetResult(&serviceInfo).Get(a.traefikURL + traefikV2ServicesURL)
	if nil != err {
		return nil, fmt.Errorf("unable to GET Traefik services info: %w", err)
	}
	if rs.StatusCode() != http.StatusOK {
		return nil, errGetHealthErr
	}

	nodesInfo := make(map[string]*NodeInfo, len(serviceInfo))

	for _, b := range serviceInfo {
		backName := b.Name[:strings.LastIndex(b.Name, "@")]
		nodesInfo[backName] = &NodeInfo{URL: b.LoadBalancer.Servers[0].URL}
	}

	return nodesInfo, nil
}

func getFirstNode(m map[string]*Server) *Server {
	for _, v := range m {
		return v
	}

	return nil
}

type serviceRepresentation struct {
	*ServiceInfo
	ServerStatus map[string]string `json:"serverStatus,omitempty"`
	Name         string            `json:"name,omitempty"`
	Provider     string            `json:"provider,omitempty"`
	Type         string            `json:"type,omitempty"`
}

// ServiceInfo holds information about a currently running service.
type ServiceInfo struct {
	LoadBalancer *ServersLoadBalancer `json:"loadBalancer,omitempty" toml:"loadBalancer,omitempty" yaml:"loadBalancer,omitempty"`
	Weighted     *WeightedRoundRobin  `json:"weighted,omitempty" toml:"weighted,omitempty" yaml:"weighted,omitempty" label:"-"`
	Mirroring    *Mirroring           `json:"mirroring,omitempty" toml:"mirroring,omitempty" yaml:"mirroring,omitempty" label:"-"`

	// Err contains all the errors that occurred during service creation.
	Err []string `json:"error,omitempty"`
	// Status reports whether the service is disabled, in a warning state, or all good (enabled).
	// If not in "enabled" state, the reason for it should be in the list of Err.
	// It is the caller's responsibility to set the initial status.
	Status string   `json:"status,omitempty"`
	UsedBy []string `json:"usedBy,omitempty"` // list of routers using that service
}

// ServersLoadBalancer holds the ServersLoadBalancer configuration.
type ServersLoadBalancer struct {
	Servers     []Server     `json:"servers,omitempty" toml:"servers,omitempty" yaml:"servers,omitempty" label-slice-as-struct:"server"`
	HealthCheck *HealthCheck `json:"healthCheck,omitempty" toml:"healthCheck,omitempty" yaml:"healthCheck,omitempty"`
}

// HealthCheck holds the HealthCheck configuration.
type HealthCheck struct {
	Scheme   string            `json:"scheme,omitempty" toml:"scheme,omitempty" yaml:"scheme,omitempty"`
	Path     string            `json:"path,omitempty" toml:"path,omitempty" yaml:"path,omitempty"`
	Port     int               `json:"port,omitempty" toml:"port,omitempty,omitzero" yaml:"port,omitempty"`
	Interval string            `json:"interval,omitempty" toml:"interval,omitempty" yaml:"interval,omitempty"`
	Timeout  string            `json:"timeout,omitempty" toml:"timeout,omitempty" yaml:"timeout,omitempty"`
	Hostname string            `json:"hostname,omitempty" toml:"hostname,omitempty" yaml:"hostname,omitempty"`
	Headers  map[string]string `json:"headers,omitempty" toml:"headers,omitempty" yaml:"headers,omitempty"`
}

// Sticky holds the sticky configuration.
type Sticky struct {
	Cookie *Cookie `json:"cookie,omitempty" toml:"cookie,omitempty" yaml:"cookie,omitempty"`
}

// Cookie holds the sticky configuration based on cookie.
type Cookie struct {
	Name     string `json:"name,omitempty" toml:"name,omitempty" yaml:"name,omitempty"`
	Secure   bool   `json:"secure,omitempty" toml:"secure,omitempty" yaml:"secure,omitempty"`
	HTTPOnly bool   `json:"httpOnly,omitempty" toml:"httpOnly,omitempty" yaml:"httpOnly,omitempty"`
}

// WeightedRoundRobin is a weighted round robin load-balancer of services.
type WeightedRoundRobin struct {
	Services []WRRService `json:"services,omitempty" toml:"services,omitempty" yaml:"services,omitempty"`
	Sticky   *Sticky      `json:"sticky,omitempty" toml:"sticky,omitempty" yaml:"sticky,omitempty"`
}

// WRRService is a reference to a service load-balanced with weighted round robin.
type WRRService struct {
	Name   string `json:"name,omitempty" toml:"name,omitempty" yaml:"name,omitempty"`
	Weight *int   `json:"weight,omitempty" toml:"weight,omitempty" yaml:"weight,omitempty"`
}

// Mirroring holds the Mirroring configuration.
type Mirroring struct {
	Service string          `json:"service,omitempty" toml:"service,omitempty" yaml:"service,omitempty"`
	Mirrors []MirrorService `json:"mirrors,omitempty" toml:"mirrors,omitempty" yaml:"mirrors,omitempty"`
}

// +k8s:deepcopy-gen=true

// MirrorService holds the MirrorService configuration.
type MirrorService struct {
	Name    string `json:"name,omitempty" toml:"name,omitempty" yaml:"name,omitempty"`
	Percent int    `json:"percent,omitempty" toml:"percent,omitempty" yaml:"percent,omitempty"`
}
