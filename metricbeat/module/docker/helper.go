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

package docker

import (
	"github.com/docker/docker/api/types/container"

	helpers "github.com/elastic/elastic-agent-autodiscover/docker"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Container is a struct representation of a container
type Container struct {
	ID     string
	Name   string
	Image  string
	Labels mapstr.M
}

// ToMapStr converts a container struct to a MapStrs
func (c *Container) ToMapStr() mapstr.M {
	m := mapstr.M{
		"container": mapstr.M{
			"id":   c.ID,
			"name": c.Name,
			"image": mapstr.M{
				"name": c.Image,
			},
			"runtime": "docker",
		},
	}

	if len(c.Labels) > 0 {
		m.Put("docker.container.labels", c.Labels)
	}
	return m
}

// NewContainer converts Docker API container to an internal structure, it applies
// dedot to container labels if dedot is true, or stores them in a nested way if it's
// false
func NewContainer(container *container.Summary, dedot bool) *Container {
	return &Container{
		ID:     container.ID,
		Name:   helpers.ExtractContainerName(container.Names),
		Labels: helpers.DeDotLabels(container.Labels, dedot),
		Image:  container.Image,
	}
}
