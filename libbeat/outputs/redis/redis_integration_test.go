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

//go:build integration

package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	_ "github.com/elastic/beats/v7/libbeat/outputs/codec/format"
	_ "github.com/elastic/beats/v7/libbeat/outputs/codec/json"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	RedisDefaultHost = "localhost"
	RedisDefaultPort = "6379"

	SRedisDefaultHost = "localhost"
	SRedisDefaultPort = "6380"
)

const (
	testBeatname    = "libbeat"
	testBeatversion = "1.2.3"
	testMetaValue   = "private"
)

func TestPublishListTCP(t *testing.T) {
	key := "test_publish_tcp"
	db := 0
	redisConfig := map[string]interface{}{
		"hosts":    []string{getRedisAddr()},
		"key":      key,
		"db":       db,
		"datatype": "list",
		"timeout":  "5s",
	}

	testPublishList(t, redisConfig)
}

func TestPublishListTLS(t *testing.T) {
	key := "test_publish_tls"
	db := 0
	redisConfig := map[string]interface{}{
		"hosts":    []string{getSRedisAddr()},
		"key":      key,
		"db":       db,
		"datatype": "list",
		"timeout":  "5s",

		// Use certificate level verification to avoid verifying the hostname when using localhost.
		"ssl.verification_mode": "certificate",
		"ssl.certificate_authorities": []string{
			"../../../testing/environments/docker/sredis/pki/tls/certs/sredis.crt",
		},
	}

	testPublishList(t, redisConfig)
}

func TestWithSchema(t *testing.T) {
	redisURL := "redis://" + getRedisAddr()
	sredisURL := "rediss://" + getSRedisAddr()

	cases := map[string]struct {
		host string
	}{
		"redis ignores ssl settings": {
			host: redisURL,
		},
		"sredis schema sends via tls": {
			host: sredisURL,
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			key := "test_publish_tls"
			db := 0
			redisConfig := map[string]interface{}{
				"hosts":    []string{test.host},
				"key":      key,
				"db":       db,
				"datatype": "list",
				"timeout":  "5s",

				// Use certificate level verification to avoid verifying the hostname when using localhost.
				"ssl.verification_mode": "certificate",
				"ssl.certificate_authorities": []string{
					"../../../testing/environments/docker/sredis/pki/tls/certs/sredis.crt",
				},
			}

			testPublishList(t, redisConfig)
		})
	}

}

func testPublishList(t *testing.T, cfg map[string]interface{}) {
	batches := 100
	batchSize := 1000
	total := batches & batchSize

	db := 0
	key, ok := cfg["key"].(string)
	if !ok {
		t.Fatalf("expected string for key, but got %T", cfg["key"])
	}

	if v, ok := cfg["db"]; ok {
		if dbValue, ok := v.(int); ok {
			db = dbValue
		} else {
			t.Fatalf("expected int for db, but got %T", v)
		}
	}

	conn, err := redis.Dial("tcp", getRedisAddr(), redis.DialDatabase(db))
	if err != nil {
		t.Fatalf("redis.Dial failed %v", err)
	}

	// delete old key if present
	defer conn.Close()
	_, err = conn.Do("DEL", key)
	require.NoError(t, err)

	out := newRedisTestingOutput(t, cfg)
	err = sendTestEvents(out, batches, batchSize)
	assert.NoError(t, err)

	results := make([][]byte, total)
	for i := range results {
		results[i], err = redis.Bytes(conn.Do("LPOP", key))
		assert.NoError(t, err)
	}

	for i, raw := range results {
		evt := struct{ Message int }{}
		err = json.Unmarshal(raw, &evt)
		assert.NoError(t, err)
		assert.Equal(t, i+1, evt.Message)
	}

	for _, raw := range results {
		validateMeta(t, raw)
	}
}

func TestPublishChannelTCP(t *testing.T) {
	db := 0
	key := "test_pubchan_tcp"
	redisConfig := map[string]interface{}{
		"hosts":    []string{getRedisAddr()},
		"key":      key,
		"db":       db,
		"datatype": "channel",
		"timeout":  "5s",
	}

	testPublishChannel(t, redisConfig)
}

func TestPublishChannelTLS(t *testing.T) {
	db := 0
	key := "test_pubchan_tls"
	redisConfig := map[string]interface{}{
		"hosts":    []string{getSRedisAddr()},
		"key":      key,
		"db":       db,
		"datatype": "channel",
		"timeout":  "5s",

		// Use certificate level verification to avoid verifying the hostname when using localhost.
		"ssl.verification_mode": "certificate",
		"ssl.certificate_authorities": []string{
			"../../../testing/environments/docker/sredis/pki/tls/certs/sredis.crt",
		},
	}

	testPublishChannel(t, redisConfig)
}

func TestPublishChannelTCPWithFormatting(t *testing.T) {
	t.Skip("format string not yet supported")
	db := 0
	key := "test_pubchan_tcp"
	redisConfig := map[string]interface{}{
		"hosts":               []string{getRedisAddr()},
		"key":                 key,
		"db":                  db,
		"datatype":            "channel",
		"timeout":             "5s",
		"codec.format.string": "%{[message]}",
	}

	testPublishChannel(t, redisConfig)
}

func testPublishChannel(t *testing.T, cfg map[string]interface{}) {
	batches := 100
	batchSize := 1000
	total := batches & batchSize

	db := 0
	key, ok := cfg["key"].(string)
	if !ok {
		t.Fatalf("expected string for key, but got %T", cfg["key"])
	}

	if v, ok := cfg["db"]; ok {
		if dbValue, ok := v.(int); ok {
			db = dbValue
		} else {
			t.Fatalf("expected int for db, but got %T", v)
		}
	}

	conn, err := redis.Dial("tcp", getRedisAddr(), redis.DialDatabase(db))
	if err != nil {
		t.Fatalf("redis.Dial failed %v", err)
	}

	// delete old key if present
	defer conn.Close()
	_, err = conn.Do("DEL", key)
	require.NoError(t, err)

	// subscribe to packetbeat channel
	psc := redis.PubSubConn{Conn: conn}
	if err := psc.Subscribe(key); err != nil {
		t.Fatal(err)
	}
	defer psc.Unsubscribe(key) //nolint:errcheck //This is a test file

	// connect and publish events
	var wg sync.WaitGroup
	var pubErr error
	out := newRedisTestingOutput(t, cfg)
	wg.Add(1)
	go func() {
		defer wg.Done()
		pubErr = sendTestEvents(out, batches, batchSize)
	}()

	// collect published events by subscription
	var messages [][]byte
	assert.NoError(t, conn.Err())
	for conn.Err() == nil {
		switch v := psc.Receive().(type) {
		case redis.Message:
			messages = append(messages, v.Data)
		case error:
			t.Error(v)
		default:
			t.Logf("received: %#v", v)
		}

		if len(messages) == total {
			break
		}
	}
	wg.Wait()

	// validate
	assert.NoError(t, pubErr)
	assert.Equal(t, total, len(messages))
	for i, raw := range messages {
		evt := struct{ Message int }{}
		if _, hasFmt := cfg["codec.format.string"]; hasFmt {
			t.Fatal("format string not yet supported")
			/*
				fmtString := fmtstr.MustCompileEvent(fmt.(string))
				expectedMessage, _ := fmtString.Run(createEvent(i + 1))
				assert.NoError(t, err)
				assert.Equal(t, string(expectedMessage), string(raw))
			*/
		} else {
			err = json.Unmarshal(raw, &evt)
			assert.NoError(t, err)
			assert.Equal(t, i+1, evt.Message)
		}
	}

	for _, raw := range messages {
		validateMeta(t, raw)
	}
}

func getEnv(name, or string) string {
	if x := os.Getenv(name); x != "" {
		return x
	}
	return or
}

func getRedisAddr() string {
	return fmt.Sprintf("%v:%v",
		getEnv("REDIS_HOST", RedisDefaultHost),
		getEnv("REDIS_PORT", RedisDefaultPort))
}

func getSRedisAddr() string {
	return fmt.Sprintf("%v:%v",
		getEnv("SREDIS_HOST", SRedisDefaultHost),
		getEnv("SREDIS_PORT", SRedisDefaultPort))
}

func newRedisTestingOutput(t *testing.T, cfg map[string]interface{}) outputs.Client {
	config, err := conf.NewConfigFrom(cfg)
	if err != nil {
		t.Fatalf("Error reading config: %v", err)
	}

	plugin := outputs.FindFactory("redis")
	if plugin == nil {
		t.Fatalf("redis output module not registered")
	}

	logger := logptest.NewTestingLogger(t, "")
	out, err := plugin(nil, beat.Info{Beat: testBeatname, Version: testBeatversion, Logger: logger}, outputs.NewNilObserver(), config)
	if err != nil {
		t.Fatalf("Failed to initialize redis output: %v", err)
	}

	client, ok := out.Clients[0].(outputs.NetworkClient)
	if !ok {
		t.Fatalf("expected outputs.NetworkClient, but got %T", out.Clients[0])
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect to redis host: %v", err)
	}

	return client
}

func sendTestEvents(out outputs.Client, batches, N int) error {
	i := 1
	for b := 0; b < batches; b++ {
		events := make([]beat.Event, N)
		for n := range events {
			events[n] = createEvent(i)
			i++
		}

		batch := outest.NewBatch(events...)
		err := out.Publish(context.Background(), batch)
		if err != nil {
			return err
		}
	}

	return nil
}

func createEvent(message int) beat.Event {
	return beat.Event{
		Timestamp: time.Now(),
		Meta: mapstr.M{
			"test": testMetaValue,
		},
		Fields: mapstr.M{"message": message},
	}
}

func validateMeta(t *testing.T, raw []byte) {
	// require metadata
	type meta struct {
		Beat    string `struct:"beat"`
		Version string `struct:"version"`
		Test    string `struct:"test"`
	}

	evt := struct {
		Meta meta `json:"@metadata"`
	}{}
	err := json.Unmarshal(raw, &evt)
	if err != nil {
		t.Errorf("failed to unmarshal meta section: %v", err)
		return
	}

	assert.Equal(t, testBeatname, evt.Meta.Beat)
	assert.Equal(t, testBeatversion, evt.Meta.Version)
	assert.Equal(t, testMetaValue, evt.Meta.Test)
}
