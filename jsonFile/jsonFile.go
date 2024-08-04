package jsonFile

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

// Aggregator holds the services URLs from Json file.
type Aggregator struct {
	services map[string]*NodeInfo
	r        *resty.Client
}

// NodeInfo embeds node-related information
type NodeInfo struct {
	URL string `json:"url"`
}

// GetInfoEndpoint returns info endpoint URL
func (ni *NodeInfo) GetInfoEndpoint() string {
	infoEndpoint, err := url.JoinPath(ni.URL, "/info")
	if nil != err {
		log.Errorf("Unable to join URL: %v", err)
	}

	return infoEndpoint
}

// GetHealthEndpoint returns health check URL
func (ni *NodeInfo) GetHealthEndpoint() string {
	healthEndpoint, err := url.JoinPath(ni.URL, "/health")
	if nil != err {
		log.Errorf("Unable to join URL: %v", err)
	}

	return healthEndpoint
}

// NewAggregator creates new aggregator from Json file
func NewAggregator(configFile string, timeout time.Duration) (*Aggregator, error) {
	services, err := loadProperties(configFile)
	if err != nil {
		return nil, err
	}

	return &Aggregator{
		r: resty.NewWithClient(&http.Client{
			Timeout: timeout,
		}),
		services: services,
	}, nil

}

// LoadServices URLs from a Json file
func loadProperties(filename string) (map[string]*NodeInfo, error) {
	file, err := os.Open(filename)
	if err != nil {
		log.Errorf("Failed to open file: %v", err)
	}
	defer file.Close()

	var byteValue []byte
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		byteValue = append(byteValue, scanner.Bytes()...)
	}
	if err := scanner.Err(); err != nil {
		log.Errorf("Failed to read file: %v", err)
	}

	var nodesInfo map[string]*NodeInfo
	if err := json.Unmarshal(byteValue, &nodesInfo); err != nil {
		log.Errorf("Failed to unmarshal Json: %v", err)
	}

	if len(nodesInfo) == 0 {
		log.Errorf("Couldn't read any service from Json file")
	}

	log.Infof("Loaded services URLs:")
	for service, info := range nodesInfo {
		log.Infof("Service: %s, URL: %s", service, info.URL)
	}

	return nodesInfo, nil
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

			return nil, errors.New("response is empty")
		}

		return rs, nil
	})
}

func (a *Aggregator) aggregate(f func(ni *NodeInfo) (interface{}, error)) map[string]interface{} {

	nodeLen := len(a.services)
	aggregated := make(map[string]interface{}, nodeLen)
	var wg sync.WaitGroup

	wg.Add(nodeLen)
	var mu sync.Mutex
	for node, info := range a.services {
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
