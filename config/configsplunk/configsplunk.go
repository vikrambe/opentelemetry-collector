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

// Package configsplunk defines the splunk HEC configuration settings.
package configsplunk

// GRPCSettings defines common settings for a gRPC configuration.
type SplunkSettings struct {
	// HEC to publish traces
	HttpEventController string `mapstructure:"http_event_controller"`

	// Token for HEC
	Token string `mapstructure:"token"`

	// Splunk index
	SplunkIndex string `mapstructure:"splunk_index"`

	// Source name
	Source string `mapstructure:"source"`

	// Source Type
	SourceType string `mapstructure:"source_type"`
}