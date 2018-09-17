package entities

import (
	"fmt"
	"strings"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/nri-mongodb/src/metrics"
)

// collectServerStatus collects serverStatus metrics
func collectServerStatus(c Collector, ms *metric.Set) error {

	// Retrieve the session for the collector
	session, err := c.GetSession()
	if err != nil {
		return fmt.Errorf("invalid session: %v", err)
	}

	// Collect and unmarshal the result
	var ss metrics.ServerStatus
	if err := session.DB("admin").Run(cmd{"serverStatus": 1}, &ss); err != nil {
		return fmt.Errorf("run serverStatus failed: %v", err)
	}

	// Insert the metrics into the metric set
	if err := ms.MarshalMetrics(ss); err != nil {
		return fmt.Errorf("marshal metrics on serverStatus failed: %v", err)
	}

	return nil
}

// collectIsMaster collects isMaster metrics. Returns a boolean which
// is true if the session is connected to a replica set
func collectIsMaster(c Collector, ms *metric.Set) (bool, error) {

	// Retrieve the session for the collector
	session, err := c.GetSession()
	if err != nil {
		return false, fmt.Errorf("invalid session: %v", err)
	}

	// Collect and unmarshal the result
	var isMaster metrics.IsMaster
	if err := session.DB("admin").Run(cmd{"isMaster": 1}, &isMaster); err != nil {
		return false, fmt.Errorf("run isMaster failed: %v", err)
	}

	// Insert the metrics into the metric set
	if err := ms.MarshalMetrics(isMaster); err != nil {
		return false, fmt.Errorf("marshal metrics on isMaster failed: %v", err)
	}

	// Return whether the node is part of a replica set and an error
	return isMaster.SetName != nil, nil
}

// collectReplGetStatus collects replica set metrics
func collectReplGetStatus(c Collector, hostname string, ms *metric.Set) error {

	// Retrieve the session for the collector
	session, err := c.GetSession()
	if err != nil {
		return fmt.Errorf("invalid session: %v", err)
	}

	// Collect and unmarshal the metrics
	var replSetStatus metrics.ReplSetGetStatus
	if err := session.DB("admin").Run(cmd{"replSetGetStatus": 1}, &replSetStatus); err != nil {
		return err
	}

	for _, member := range replSetStatus.Members {
		if !strings.HasPrefix(*member.Name, hostname) { // TODO ensure that the member name will always be the hostname
			continue
		}
		logError(ms.MarshalMetrics(member), "Marshal metrics on replSetGetStatus failed: %v")
	}

	return nil

}

// collectReplGetConfig collects replica set metrics
func collectReplGetConfig(c Collector, hostname string, ms *metric.Set) error {

	// Retrieve the session for the collector
	session, err := c.GetSession()
	if err != nil {
		return fmt.Errorf("invalid session: %v", err)
	}

	// Collect and unmarshal the metrics
	var replSetConfig metrics.ReplSetGetConfig
	if err := session.DB("admin").Run(cmd{"replSetGetConfig": 1}, &replSetConfig); err != nil {
		return err
	}

	for _, member := range replSetConfig.Config.Members {
		if !strings.HasPrefix(*member.Host, hostname) { // TODO ensure that the member name will always be the hostname
			continue
		}
		logError(ms.MarshalMetrics(member), "Marshal metrics on replSetGetConfig failed: %v")
	}

	return nil

}

// collectTop collects top metrics
func collectTop(c Collector) error {
	session, err := c.GetSession()
	if err != nil {
		return fmt.Errorf("invalid session: %v", err)
	}

	e, err := c.GetEntity()
	if err != nil {
		return fmt.Errorf("invalid entity: %v", err)
	}

	var topMetrics metrics.Top
	if err := session.DB("admin").Run(cmd{"top": 1}, &topMetrics); err != nil {
		return fmt.Errorf("run serverStatus failed: %v", err)
	}

	for key, collectionStats := range topMetrics.Totals {
		splitKey := strings.SplitN(key, ".", 2)
		databaseName := splitKey[0]
		collectionName := splitKey[1]

		ms := e.NewMetricSet("MongodTopSample",
			metric.Attribute{Key: "displayName", Value: e.Metadata.Name},
			metric.Attribute{Key: "entityName", Value: fmt.Sprintf("%s:%s", e.Metadata.Namespace, e.Metadata.Name)},
			metric.Attribute{Key: "database", Value: databaseName},
			metric.Attribute{Key: "collection", Value: collectionName},
		)

		logError(ms.MarshalMetrics(collectionStats), "Marshal metrics on top failed: %v")

	}

	return nil
}

// collectCollStats collects collStats
func collectCollStats(c *collectionCollector, ms *metric.Set) error {

	// Ignore system collections as they're likely not wanted and probably don't have permission anyway
	if strings.HasPrefix(c.name, "system.") {
		return nil
	}

	session, err := c.GetSession()
	if err != nil {
		return fmt.Errorf("invalid session: %v", err)
	}

	var collStats metrics.CollStats
	if err := session.DB(c.db).Run(cmd{"collStats": c.name}, &collStats); err != nil {
		return fmt.Errorf("run collStats failed: %v", err)
	}

	return ms.MarshalMetrics(collStats)
}

// collectDbStats collects dbStats
func collectDbStats(c *databaseCollector, ms *metric.Set) error {
	var dbStats metrics.DbStats
	if err := c.session.DB(c.name).Run(cmd{"dbStats": 1}, &dbStats); err != nil {
		return fmt.Errorf("run dbStats failed: %s", err)
	}

	return ms.MarshalMetrics(dbStats)
}