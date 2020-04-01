// Copyright  OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
)

type TestExporterFactory struct {
	name string
}

// Type gets the type of the Exporter config created by this factory.
func (f *TestExporterFactory) Type() string {
	return f.name
}

// CreateDefaultConfig creates the default configuration for the Exporter.
func (f *TestExporterFactory) CreateDefaultConfig() configmodels.Exporter {
	return nil
}

// CreateTraceExporter creates a trace exporter based on this config.
func (f *TestExporterFactory) CreateTraceExporter(logger *zap.Logger, cfg configmodels.Exporter) (TraceExporterOld, error) {
	return nil, nil
}

// CreateMetricsExporter creates a metrics exporter based on this config.
func (f *TestExporterFactory) CreateMetricsExporter(logger *zap.Logger, cfg configmodels.Exporter) (MetricsExporterOld, error) {
	return nil, nil
}

func TestBuildExporters(t *testing.T) {
	type testCase struct {
		in  []ExporterFactoryBase
		out map[string]ExporterFactoryBase
		err bool
	}

	testCases := []testCase{
		{
			in: []ExporterFactoryBase{
				&TestExporterFactory{"exp1"},
				&TestExporterFactory{"exp2"},
			},
			out: map[string]ExporterFactoryBase{
				"exp1": &TestExporterFactory{"exp1"},
				"exp2": &TestExporterFactory{"exp2"},
			},
			err: false,
		},
		{
			in: []ExporterFactoryBase{
				&TestExporterFactory{"exp1"},
				&TestExporterFactory{"exp1"},
			},
			err: true,
		},
	}

	for _, c := range testCases {
		out, err := MakeExporterFactoryMap(c.in...)
		if c.err {
			assert.NotNil(t, err)
			continue
		}
		assert.Nil(t, err)
		assert.Equal(t, c.out, out)
	}
}
