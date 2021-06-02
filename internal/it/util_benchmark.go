/*
 * Copyright (c) 2008-2021, Hazelcast, Inc. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License")
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package it

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/hazelcast/hazelcast-go-client/logger"

	hz "github.com/hazelcast/hazelcast-go-client"
)

const EnvWarmupCount = "WARMUPS"

func Benchmarker(b *testing.B, f func(b *testing.B, config *hz.Config)) {
	BenchmarkerWithConfigBuilder(b, nil, f)
}

func BenchmarkerWithConfigBuilder(b *testing.B, configCallback func(*hz.Config), f func(b *testing.B, config *hz.Config)) {
	ensureRemoteController(true)
	runner := func(b *testing.B, smart bool) {
		config := defaultTestCluster.DefaultConfig()
		if configCallback != nil {
			configCallback(&config)
		}
		config.LoggerConfig.Level = logger.ErrorLevel
		config.ClusterConfig.SmartRouting = smart
		b.ResetTimer()
		f(b, &config)
	}
	if SmartEnabled() {
		b.Run("Smart Client", func(b *testing.B) {
			runner(b, true)
		})
	}
	if NonSmartEnabled() {
		b.Run("Non-Smart Client", func(b *testing.B) {
			runner(b, false)
		})
	}
}

func MapBenchmarker(t *testing.B, fixture func(m *hz.Map), f func(t *testing.B, m *hz.Map)) {
	configCallback := func(cb *hz.Config) {
	}
	MapBenchmarkerWithConfigBuilder(t, configCallback, fixture, f)
}

func MapBenchmarkerWithConfigBuilder(t *testing.B, cbCallback func(*hz.Config), fixture func(m *hz.Map), f func(t *testing.B, m *hz.Map)) {
	makeMapName := func() string {
		return fmt.Sprintf("bm-map-%d", rand.Int())
	}
	MapBenchmarkerWithConfigAndName(t, makeMapName, cbCallback, fixture, f)
}

func MapBenchmarkerWithConfigAndName(b *testing.B, makeMapName func() string, cbCallback func(config *hz.Config), fixture func(m *hz.Map), f func(b *testing.B, m *hz.Map)) {
	var (
		client *hz.Client
		m      *hz.Map
	)
	ensureRemoteController(true)
	warmups := warmupCount()
	if warmups > 0 {
		b.Logf("Warmups: %d", warmups)
	}
	runner := func(b *testing.B, smart bool) {
		client, m = getMap(makeMapName(), cbCallback, smart)
		defer func() {
			m.EvictAll()
			if err := client.Shutdown(); err != nil {
				b.Logf("Test warning, client not shutdown: %s", err.Error())
			}
		}()
		if fixture != nil {
			fixture(m)
		}
		b.ResetTimer()
		f(b, m)
	}
	warmJvmUp := func() {
		client, m := getMap(makeMapName(), cbCallback, true)
		for i := 0; i < warmups; i++ {
			f(b, m)
		}
		m.EvictAll()
		client.Shutdown()
	}
	if SmartEnabled() {
		b.Run("Smart Client", func(b *testing.B) {
			warmJvmUp()
			runner(b, true)
		})
	}
	if NonSmartEnabled() {
		b.Run("Non-Smart Client", func(b *testing.B) {
			warmJvmUp()
			runner(b, false)
		})
	}
}

func warmupCount() int {
	if s := os.Getenv(EnvWarmupCount); s != "" {
		if i, err := strconv.ParseInt(s, 10, 32); err != nil {
			panic(err)
		} else {
			return int(i)
		}
	}
	return 3
}

func getMap(mapName string, configCallback func(*hz.Config), smart bool) (*hz.Client, *hz.Map) {
	config := defaultTestCluster.DefaultConfig()
	if configCallback != nil {
		configCallback(&config)
	}
	config.ClusterConfig.SmartRouting = smart
	config.LoggerConfig.Level = logger.ErrorLevel
	return GetClientMapWithConfig(mapName, &config)
}