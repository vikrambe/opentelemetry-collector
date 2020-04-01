// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package collector handles the command-line, configuration, and runs the OC collector.
package service

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/defaults"
	"github.com/open-telemetry/opentelemetry-collector/testutils"
)

func TestApplication_Start(t *testing.T) {
	factories, err := defaults.Components()
	require.NoError(t, err)

	app, err := New(Parameters{Factories: factories, ApplicationStartInfo: ApplicationStartInfo{}})
	require.NoError(t, err)
	assert.Equal(t, app.rootCmd, app.Command())

	const testPrefix = "a_test"
	metricsPort := testutils.GetAvailablePort(t)
	app.rootCmd.SetArgs([]string{
		"--config=testdata/otelcol-config.yaml",
		"--metrics-port=" + strconv.FormatUint(uint64(metricsPort), 10),
		"--metrics-prefix=" + testPrefix,
		"--add-instance-id=true",
	})

	appDone := make(chan struct{})
	go func() {
		defer close(appDone)
		assert.NoError(t, app.Start())
	}()

	<-app.readyChan

	require.True(t, isAppAvailable(t, "http://localhost:13133"))

	assertMetricsPrefix(t, testPrefix, metricsPort)

	close(app.stopTestChan)
	<-appDone
}

// isAppAvailable checks if the healthcheck server at the given endpoint is
// returning `available`.
func isAppAvailable(t *testing.T, healthCheckEndPoint string) bool {
	client := &http.Client{}
	resp, err := client.Get(healthCheckEndPoint)
	require.NoError(t, err)

	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func assertMetricsPrefix(t *testing.T, prefix string, metricsPort uint16) {
	client := &http.Client{}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/metrics", metricsPort))
	require.NoError(t, err)

	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)

	for {
		s, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}

		require.NoError(t, err)
		if len(s) == 0 || s[0] == '#' {
			// Skip this line since it is not a metric.
			continue
		}

		// require is used here so test fails with a single message.
		require.True(
			t,
			strings.HasPrefix(s, prefix),
			"expected prefix %q but string starts with %q",
			prefix,
			s[:len(prefix)+1]+"...")
	}
}

func TestApplication_setupExtensions(t *testing.T) {
	exampleExtensionFactory := &config.ExampleExtensionFactory{}
	exampleExtensionConfig := &config.ExampleExtension{
		ExtensionSettings: configmodels.ExtensionSettings{
			TypeVal: exampleExtensionFactory.Type(),
			NameVal: exampleExtensionFactory.Type(),
		},
	}

	badExtensionFactory := &badExtensionFactory{}
	badExtensionFactoryConfig := &configmodels.ExtensionSettings{
		TypeVal: "bf",
		NameVal: "bf",
	}

	tests := []struct {
		name       string
		factories  config.Factories
		config     *configmodels.Config
		wantErrMsg string
	}{
		{
			name: "extension_not_configured",
			config: &configmodels.Config{
				Service: configmodels.Service{
					Extensions: []string{
						"myextension",
					},
				},
			},
			wantErrMsg: "extension \"myextension\" is not configured",
		},
		{
			name: "missing_extension_factory",
			config: &configmodels.Config{
				Extensions: map[string]configmodels.Extension{
					exampleExtensionFactory.Type(): exampleExtensionConfig,
				},
				Service: configmodels.Service{
					Extensions: []string{
						exampleExtensionFactory.Type(),
					},
				},
			},
			wantErrMsg: "extension factory for type \"exampleextension\" is not configured",
		},
		{
			name: "error_on_create_extension",
			factories: config.Factories{
				Extensions: map[string]component.ExtensionFactory{
					exampleExtensionFactory.Type(): exampleExtensionFactory,
				},
			},
			config: &configmodels.Config{
				Extensions: map[string]configmodels.Extension{
					exampleExtensionFactory.Type(): exampleExtensionConfig,
				},
				Service: configmodels.Service{
					Extensions: []string{
						exampleExtensionFactory.Type(),
					},
				},
			},
			wantErrMsg: "failed to create extension \"exampleextension\": cannot create \"exampleextension\" extension type",
		},
		{
			name: "bad_factory",
			factories: config.Factories{
				Extensions: map[string]component.ExtensionFactory{
					badExtensionFactory.Type(): badExtensionFactory,
				},
			},
			config: &configmodels.Config{
				Extensions: map[string]configmodels.Extension{
					badExtensionFactory.Type(): badExtensionFactoryConfig,
				},
				Service: configmodels.Service{
					Extensions: []string{
						badExtensionFactory.Type(),
					},
				},
			},
			wantErrMsg: "factory for \"bf\" produced a nil extension",
		},
	}

	nopLogger := zap.NewNop()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &Application{
				logger:    nopLogger,
				factories: tt.factories,
				config:    tt.config,
			}

			err := app.setupExtensions()

			if tt.wantErrMsg == "" {
				assert.NoError(t, err)
				assert.Equal(t, 1, len(app.extensions))
				assert.NotNil(t, app.extensions[0])
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErrMsg, err.Error())
				assert.Equal(t, 0, len(app.extensions))
			}
		})
	}
}

// badExtensionFactory is a factory that returns no error but returns a nil object.
type badExtensionFactory struct{}

func (b badExtensionFactory) Type() string {
	return "bf"
}

func (b badExtensionFactory) CreateDefaultConfig() configmodels.Extension {
	return &configmodels.ExtensionSettings{}
}

func (b badExtensionFactory) CreateExtension(
	logger *zap.Logger,
	cfg configmodels.Extension,
) (component.ServiceExtension, error) {
	return nil, nil
}
