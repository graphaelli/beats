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

package stats

import (
	"encoding/json"
	"fmt"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/nats/util"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var (
	moduleSchema = s.Schema{
		"server": s.Object{
			"id":   c.Str("server_id"),
			"time": c.Str("now"),
		},
	}
	httpReqStatsSchema = s.Schema{
		"root_uri":     c.Int("/", s.Optional),
		"connz_uri":    c.Int("/connz", s.Optional),
		"routez_uri":   c.Int("/routez", s.Optional),
		"subsz_uri":    c.Int("/subsz", s.Optional),
		"varz_uri":     c.Int("/varz", s.Optional),
		"jsz_uri":      c.Int("/jsz", s.Optional),
		"accountz_uri": c.Int("/accountz", s.Optional),
		"accstatz_uri": c.Int("/accstatz", s.Optional),
		"gatewayz_uri": c.Int("/gatewayz", s.Optional),
		"healthz_uri":  c.Int("/healthz", s.Optional),
		"leafz_uri":    c.Int("/leafz", s.Optional),
	}
	statsSchema = s.Schema{
		"server_name": c.Str("server_name", s.Optional),
		"version":     c.Str("version", s.Optional),
		"uptime":      c.Str("uptime"),
		"mem": s.Object{
			"bytes": c.Int("mem"),
		},
		"cores":             c.Int("cores"),
		"cpu":               c.Float("cpu"),
		"total_connections": c.Int("total_connections"),
		"remotes":           c.Int("remotes"),
		"in": s.Object{
			"messages": c.Int("in_msgs"),
			"bytes":    c.Int("in_bytes"),
		},
		"out": s.Object{
			"messages": c.Int("out_msgs"),
			"bytes":    c.Int("out_bytes"),
		},
		"slow_consumers": c.Int("slow_consumers"),
		"http_req_stats": c.Dict("http_req_stats", httpReqStatsSchema),
	}
)

func eventMapping(r mb.ReporterV2, content []byte) error {
	var metricsetMetrics mapstr.M
	var inInterface map[string]interface{}

	err := json.Unmarshal(content, &inInterface)
	if err != nil {
		return fmt.Errorf("failure parsing Nats stats API response: %w", err)
	}
	metricsetMetrics, err = statsSchema.Apply(inInterface)
	if err != nil {
		return fmt.Errorf("failure applying stats schema: %w", err)
	}

	err = util.UpdateDuration(metricsetMetrics, "uptime")
	if err != nil {
		return fmt.Errorf("failure updating uptime key: %w", err)
	}

	d, err := metricsetMetrics.GetValue("http_req_stats")
	if err != nil {
		return fmt.Errorf("failure retrieving http_req_stats key: %w", err)
	}
	httpStats, ok := d.(mapstr.M)
	if !ok {
		return fmt.Errorf("failure casting http_req_stats to common.Mapstr")
	}
	err = metricsetMetrics.Delete("http_req_stats")
	if err != nil {
		return fmt.Errorf("failure deleting http_req_stats key: %w", err)

	}

	httpMetrics := []string{
		"root",
		"connz",
		"routez",
		"subsz",
		"varz",
		"jsz",
		"accountz",
		"accstatz",
		"gatewayz",
		"healthz",
		"leafz",
	}

	metricsetMetrics["http"] = mapstr.M{
		"req_stats": mapstr.M{
			"uri": mapstr.M{},
		},
	}

	for _, name := range httpMetrics {
		key := fmt.Sprintf("%s_uri", name)
		if httpStats[key] != nil {
			metricsetMetrics.Put(fmt.Sprintf("http.req_stats.uri.%s", name), httpStats[key])
		}
	}

	cpu, err := metricsetMetrics.GetValue("cpu")
	if err != nil {
		return fmt.Errorf("failure retrieving cpu key: %w", err)
	}
	cpuUtil, ok := cpu.(float64)
	if !ok {
		return fmt.Errorf("failure casting cpu to float64")
	}
	_, err = metricsetMetrics.Put("cpu", cpuUtil/100.0)
	if err != nil {
		return fmt.Errorf("failure updating cpu key: %w", err)
	}
	moduleMetrics, err := moduleSchema.Apply(inInterface)
	if err != nil {
		return fmt.Errorf("failure applying module schema: %w", err)
	}
	timestamp, err := util.GetNatsTimestamp(moduleMetrics)
	if err != nil {
		return fmt.Errorf("failure parsing server timestamp: %w", err)
	}
	evt := mb.Event{
		MetricSetFields: metricsetMetrics,
		ModuleFields:    moduleMetrics,
		Timestamp:       timestamp,
	}
	r.Event(evt)
	return nil
}
