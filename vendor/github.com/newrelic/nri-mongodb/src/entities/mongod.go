package entities

import (
	"errors"
	"fmt"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-mongodb/src/connection"
)

// mongodCollector is a storage struct with all the information needed
// to collect metrics and inventory for a mongod
type mongodCollector struct {
	hostCollector
}

// GetEntity creates or returns an entity for the mongod
func (c *mongodCollector) GetEntity() (*integration.Entity, error) {
	if i := c.GetIntegration(); i != nil {
		return i.Entity(c.name, "mongod")
	}

	return nil, errors.New("nil integration")
}

// CollectInventory collects inventory
func (c *mongodCollector) CollectInventory() {
	e, err := c.GetEntity()
	if err != nil {
		log.Error("Failed to create mongod entity: %v", err)
		return
	}
	c.collectInventory(e)
}

// CollectMetrics sets all the metrics for a mongod
func (c *mongodCollector) CollectMetrics() {
	e, err := c.GetEntity()
	if logError(err, "Failed to create mongod entity: %v") {
		return
	}

	ms := e.NewMetricSet("MongodSample",
		metric.Attribute{Key: "displayName", Value: e.Metadata.Name},
		metric.Attribute{Key: "entityName", Value: fmt.Sprintf("%s:%s", e.Metadata.Namespace, e.Metadata.Name)},
	)

	isReplSet, err := collectIsMaster(c, ms)
	logError(err, "Collect is master failed: %v")

	if isReplSet {
		logError(collectReplGetStatus(c, e.Metadata.Name, ms), "Get ReplSet status failed: %v")
		logError(collectReplGetConfig(c, e.Metadata.Name, ms), "Get ReplSet config failed: %v")
	}

	logError(collectServerStatus(c, ms), "Collect server status failed: %v")
	logError(collectTop(c), "Collect top failed: %v")
}

// GetMongods returns an array of MongodCollectors to collect
func GetMongods(session connection.Session, shardHostString string, integration *integration.Integration) ([]Collector, error) {
	hostPorts, _ := parseReplicaSetString(shardHostString)

	mongodCollectors := make([]Collector, 0, len(hostPorts))
	for _, hostPort := range hostPorts {
		mongodSession, err := session.New(hostPort.Host, hostPort.Port)
		if err != nil {
			log.Error("Failed to connected to mongod server %s: %v", hostPort.Host, err)
			continue
		}

		newMongodCollector := &mongodCollector{
			hostCollector{
				defaultCollector{
					hostPort.Host,
					integration,
					mongodSession,
				},
			},
		}
		mongodCollectors = append(mongodCollectors, newMongodCollector)
	}

	return mongodCollectors, nil
}
