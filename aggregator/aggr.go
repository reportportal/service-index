package aggregator

type (
	//Aggregator collects information from all available services
	Aggregator interface {
		//AggregateInfo collects information from info endpoints
		AggregateInfo() map[string]interface{}

		//AggregateHealth aggregates information from health endpoints
		AggregateHealth() map[string]interface{}
	}
)
