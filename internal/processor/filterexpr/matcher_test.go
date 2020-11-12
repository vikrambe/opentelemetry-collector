// Copyright The OpenTelemetry Authors
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

package filterexpr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestCompileExprError(t *testing.T) {
	_, err := NewMatcher("")
	require.Error(t, err)
}

func TestRunExprError(t *testing.T) {
	matcher, err := NewMatcher("foo")
	require.NoError(t, err)
	matched, _ := matcher.match(env{})
	require.False(t, matched)
}

func TestUnknownDataType(t *testing.T) {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.InitEmpty()
	m.SetName("my.metric")
	m.SetDataType(-1)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.False(t, matched)
}

func TestNilIntGauge(t *testing.T) {
	dataType := pdata.MetricDataTypeIntGauge
	testNilValue(t, dataType)
}

func TestNilDoubleGauge(t *testing.T) {
	dataType := pdata.MetricDataTypeDoubleGauge
	testNilValue(t, dataType)
}

func TestNilDoubleSum(t *testing.T) {
	dataType := pdata.MetricDataTypeDoubleSum
	testNilValue(t, dataType)
}

func TestNilIntSum(t *testing.T) {
	dataType := pdata.MetricDataTypeIntSum
	testNilValue(t, dataType)
}

func TestNilIntHistogram(t *testing.T) {
	dataType := pdata.MetricDataTypeIntHistogram
	testNilValue(t, dataType)
}

func TestNilDoubleHistogram(t *testing.T) {
	dataType := pdata.MetricDataTypeDoubleHistogram
	testNilValue(t, dataType)
}

func testNilValue(t *testing.T, dataType pdata.MetricDataType) {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.InitEmpty()
	m.SetName("my.metric")
	m.SetDataType(dataType)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.False(t, matched)
}

func TestIntGaugeNilDataPoint(t *testing.T) {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.InitEmpty()
	m.SetName("my.metric")
	m.SetDataType(pdata.MetricDataTypeIntGauge)
	gauge := m.IntGauge()
	gauge.InitEmpty()
	dps := gauge.DataPoints()
	pt := pdata.NewIntDataPoint()
	dps.Append(pt)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.False(t, matched)
}

func TestDoubleGaugeNilDataPoint(t *testing.T) {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.InitEmpty()
	m.SetName("my.metric")
	m.SetDataType(pdata.MetricDataTypeDoubleGauge)
	gauge := m.DoubleGauge()
	gauge.InitEmpty()
	dps := gauge.DataPoints()
	pt := pdata.NewDoubleDataPoint()
	dps.Append(pt)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.False(t, matched)
}

func TestDoubleSumNilDataPoint(t *testing.T) {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.InitEmpty()
	m.SetName("my.metric")
	m.SetDataType(pdata.MetricDataTypeDoubleSum)
	sum := m.DoubleSum()
	sum.InitEmpty()
	dps := sum.DataPoints()
	pt := pdata.NewDoubleDataPoint()
	dps.Append(pt)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.False(t, matched)
}

func TestIntSumNilDataPoint(t *testing.T) {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.InitEmpty()
	m.SetName("my.metric")
	m.SetDataType(pdata.MetricDataTypeIntSum)
	sum := m.IntSum()
	sum.InitEmpty()
	dps := sum.DataPoints()
	pt := pdata.NewIntDataPoint()
	dps.Append(pt)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.False(t, matched)
}

func TestIntHistogramNilDataPoint(t *testing.T) {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.InitEmpty()
	m.SetName("my.metric")
	m.SetDataType(pdata.MetricDataTypeIntHistogram)
	h := m.IntHistogram()
	h.InitEmpty()
	dps := h.DataPoints()
	pt := pdata.NewIntHistogramDataPoint()
	dps.Append(pt)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.False(t, matched)
}

func TestDoubleHistogramNilDataPoint(t *testing.T) {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.InitEmpty()
	m.SetName("my.metric")
	m.SetDataType(pdata.MetricDataTypeDoubleHistogram)
	h := m.DoubleHistogram()
	h.InitEmpty()
	dps := h.DataPoints()
	pt := pdata.NewDoubleHistogramDataPoint()
	dps.Append(pt)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.False(t, matched)
}

func TestMatchIntGaugeByMetricName(t *testing.T) {
	expression := `MetricName == 'my.metric'`
	assert.True(t, testMatchIntGauge(t, "my.metric", expression, nil))
}

func TestNonMatchIntGaugeByMetricName(t *testing.T) {
	expression := `MetricName == 'my.metric'`
	assert.False(t, testMatchIntGauge(t, "foo.metric", expression, nil))
}

func TestNonMatchIntGaugeDataPointByMetricAndHasLabel(t *testing.T) {
	expression := `MetricName == 'my.metric' && HasLabel("foo")`
	assert.False(t, testMatchIntGauge(t, "foo.metric", expression, nil))
}

func TestMatchIntGaugeDataPointByMetricAndHasLabel(t *testing.T) {
	expression := `MetricName == 'my.metric' && HasLabel("foo")`
	assert.True(t, testMatchIntGauge(t, "my.metric", expression, map[string]string{"foo": ""}))
}

func TestMatchIntGaugeDataPointByMetricAndLabelValue(t *testing.T) {
	expression := `MetricName == 'my.metric' && Label("foo") == "bar"`
	assert.False(t, testMatchIntGauge(t, "my.metric", expression, map[string]string{"foo": ""}))
}

func TestNonMatchIntGaugeDataPointByMetricAndLabelValue(t *testing.T) {
	expression := `MetricName == 'my.metric' && Label("foo") == "bar"`
	assert.False(t, testMatchIntGauge(t, "my.metric", expression, map[string]string{"foo": ""}))
}

func testMatchIntGauge(t *testing.T, metricName, expression string, lbls map[string]string) bool {
	matcher, err := NewMatcher(expression)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.InitEmpty()
	m.SetName(metricName)
	m.SetDataType(pdata.MetricDataTypeIntGauge)
	gauge := m.IntGauge()
	gauge.InitEmpty()
	dps := gauge.DataPoints()
	pt := pdata.NewIntDataPoint()
	pt.InitEmpty()
	if lbls != nil {
		pt.LabelsMap().InitFromMap(lbls)
	}
	dps.Append(pt)
	match, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	return match
}

func TestMatchIntGaugeDataPointByMetricAndSecondPointLabelValue(t *testing.T) {
	matcher, err := NewMatcher(
		`MetricName == 'my.metric' && Label("baz") == "glarch"`,
	)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.InitEmpty()
	m.SetName("my.metric")
	m.SetDataType(pdata.MetricDataTypeIntGauge)
	gauge := m.IntGauge()
	gauge.InitEmpty()
	dps := gauge.DataPoints()

	pt1 := pdata.NewIntDataPoint()
	pt1.InitEmpty()
	pt1.LabelsMap().Insert("foo", "bar")
	dps.Append(pt1)

	pt2 := pdata.NewIntDataPoint()
	pt2.InitEmpty()
	pt2.LabelsMap().Insert("baz", "glarch")
	dps.Append(pt2)

	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	assert.True(t, matched)
}

func TestMatchDoubleGaugeByMetricName(t *testing.T) {
	assert.True(t, testMatchDoubleGauge(t, "my.metric"))
}

func TestNonMatchDoubleGaugeByMetricName(t *testing.T) {
	assert.False(t, testMatchDoubleGauge(t, "foo.metric"))
}

func testMatchDoubleGauge(t *testing.T, metricName string) bool {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.InitEmpty()
	m.SetName(metricName)
	m.SetDataType(pdata.MetricDataTypeDoubleGauge)
	gauge := m.DoubleGauge()
	gauge.InitEmpty()
	dps := gauge.DataPoints()
	pt := pdata.NewDoubleDataPoint()
	pt.InitEmpty()
	dps.Append(pt)
	match, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	return match
}

func TestMatchDoubleSumByMetricName(t *testing.T) {
	assert.True(t, matchDoubleSum(t, "my.metric"))
}

func TestNonMatchDoubleSumByMetricName(t *testing.T) {
	assert.False(t, matchDoubleSum(t, "foo.metric"))
}

func matchDoubleSum(t *testing.T, metricName string) bool {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.InitEmpty()
	m.SetName(metricName)
	m.SetDataType(pdata.MetricDataTypeDoubleSum)
	sum := m.DoubleSum()
	sum.InitEmpty()
	dps := sum.DataPoints()
	pt := pdata.NewDoubleDataPoint()
	pt.InitEmpty()
	dps.Append(pt)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	return matched
}

func TestMatchIntSumByMetricName(t *testing.T) {
	assert.True(t, matchIntSum(t, "my.metric"))
}

func TestNonMatchIntSumByMetricName(t *testing.T) {
	assert.False(t, matchIntSum(t, "foo.metric"))
}

func matchIntSum(t *testing.T, metricName string) bool {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.InitEmpty()
	m.SetName(metricName)
	m.SetDataType(pdata.MetricDataTypeIntSum)
	sum := m.IntSum()
	sum.InitEmpty()
	dps := sum.DataPoints()
	pt := pdata.NewIntDataPoint()
	pt.InitEmpty()
	dps.Append(pt)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	return matched
}

func TestMatchIntHistogramByMetricName(t *testing.T) {
	assert.True(t, matchIntHistogram(t, "my.metric"))
}

func TestNonMatchIntHistogramByMetricName(t *testing.T) {
	assert.False(t, matchIntHistogram(t, "foo.metric"))
}

func matchIntHistogram(t *testing.T, metricName string) bool {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.InitEmpty()
	m.SetName(metricName)
	m.SetDataType(pdata.MetricDataTypeIntHistogram)
	sum := m.IntHistogram()
	sum.InitEmpty()
	dps := sum.DataPoints()
	pt := pdata.NewIntHistogramDataPoint()
	pt.InitEmpty()
	dps.Append(pt)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	return matched
}

func TestMatchDoubleHistogramByMetricName(t *testing.T) {
	assert.True(t, matchDoubleHistogram(t, "my.metric"))
}

func TestNonMatchDoubleHistogramByMetricName(t *testing.T) {
	assert.False(t, matchDoubleHistogram(t, "foo.metric"))
}

func matchDoubleHistogram(t *testing.T, metricName string) bool {
	matcher, err := NewMatcher(`MetricName == 'my.metric'`)
	require.NoError(t, err)
	m := pdata.NewMetric()
	m.InitEmpty()
	m.SetName(metricName)
	m.SetDataType(pdata.MetricDataTypeDoubleHistogram)
	sum := m.DoubleHistogram()
	sum.InitEmpty()
	dps := sum.DataPoints()
	pt := pdata.NewDoubleHistogramDataPoint()
	pt.InitEmpty()
	dps.Append(pt)
	matched, err := matcher.MatchMetric(m)
	assert.NoError(t, err)
	return matched
}
