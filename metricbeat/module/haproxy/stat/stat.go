// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package stat

import (
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/haproxy"
)

const (
	statsMethod = "stat"
)

// init registers the haproxy stat MetricSet.
func init() {
	mb.Registry.MustAddMetricSet("haproxy", statsMethod, New,
		mb.WithHostParser(haproxy.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for haproxy stats.
type MetricSet struct {
	mb.BaseMetricSet
}

// New creates a new haproxy stat MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{BaseMetricSet: base}, nil
}

// Fetch methods returns a list of stats metrics.
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	hapc, err := haproxy.NewHaproxyClient(m.HostData().URI, m.BaseMetricSet)
	if err != nil {
		return fmt.Errorf("failed creating haproxy client: %w", err)
	}

	res, err := hapc.GetStat()
	if err != nil {
		return fmt.Errorf("failed fetching haproxy stat: %w", err)
	}

	eventMapping(res, reporter)
	return nil
}
