package carbon

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInstanceSpecsParsing_MalformedRows verifies that the embedded instance
// spec parser keeps valid rows when the same CSV contains bad ones.
//
// The parser is exercised through GetInstanceSpec. The embedded CSV and its
// sync.Once are swapped for the test and restored afterward, so the real
// data is unchanged for later tests.
//
// Covered rows:
//  1. A short row (missing columns) logs a skip warning and is dropped.
//  2. A later valid row, including a European decimal, is still loaded.
//  3. An empty instance type, a non-numeric vCPU, and negative or inverted
//     watts are skipped without being stored.
func TestInstanceSpecsParsing_MalformedRows(t *testing.T) {
	header := strings.Join(repeat("h", colMaxWatts+1), ",")
	csvData := strings.Join([]string{
		header,
		instanceSpecRow("good", "2", "1.5", "3.0"),
		"only,two",
		instanceSpecRow("also", "4", "1,25", "2,50"),
		instanceSpecRow("", "2", "1.5", "3.0"),
		instanceSpecRow("badcpu", "nope", "1.5", "3.0"),
		instanceSpecRow("negative", "2", "-1", "3.0"),
		instanceSpecRow("inverted", "2", "5", "1"),
	}, "\n")

	logs := captureCarbonLogs(t)
	swapInstanceSpecs(t, csvData)

	spec, ok := GetInstanceSpec("good")
	require.True(t, ok)
	assert.Equal(t, 2, spec.VCPUCount)
	assert.InDelta(t, 1.5, spec.MinWatts, 1e-9)
	assert.InDelta(t, 3.0, spec.MaxWatts, 1e-9)

	also, ok := GetInstanceSpec("also")
	require.True(t, ok)
	assert.Equal(t, 4, also.VCPUCount)
	assert.InDelta(t, 1.25, also.MinWatts, 1e-9)
	assert.InDelta(t, 2.50, also.MaxWatts, 1e-9)

	for _, missing := range []string{"", "badcpu", "negative", "inverted", "only"} {
		_, found := GetInstanceSpec(missing)
		assert.False(t, found, "row %q should be skipped", missing)
	}
	assert.Contains(t, logs.String(), "skipping malformed CCF instance specs CSV row")
}

// TestGPUSpecsParsing_MalformedRows verifies that the GPU spec parser keeps
// valid rows when the CSV also has a short row, an empty instance type, a
// bad GPU count, or a negative TDP.
func TestGPUSpecsParsing_MalformedRows(t *testing.T) {
	csvData := strings.Join([]string{
		"instance_type,gpu_model,gpu_count,tdp_per_gpu_watts",
		"p4d.24xlarge,A100,8,400",
		"short",
		",A100,8,400",
		"badcount,A100,nope,400",
		"negative,A100,1,-5",
		"g4dn.xlarge,T4,1,70",
	}, "\n")

	logs := captureCarbonLogs(t)
	swapGPUSpecs(t, csvData)

	spec, ok := GetGPUSpec("p4d.24xlarge")
	require.True(t, ok)
	assert.Equal(t, "A100", spec.GPUModel)
	assert.Equal(t, 8, spec.GPUCount)
	assert.InDelta(t, 400.0, spec.TDPPerGPU, 1e-9)

	g4, ok := GetGPUSpec("g4dn.xlarge")
	require.True(t, ok)
	assert.Equal(t, 1, g4.GPUCount)

	for _, missing := range []string{"", "badcount", "negative", "short"} {
		_, found := GetGPUSpec(missing)
		assert.False(t, found, "row %q should be skipped", missing)
	}
	assert.Contains(t, logs.String(), "skipping malformed GPU specs CSV row")
}

// TestStorageSpecsParsing_MalformedRows verifies that the storage spec parser
// keeps valid rows when the CSV also has a short row, a missing service or
// class, a replication factor below 1, or a non-positive power coefficient.
func TestStorageSpecsParsing_MalformedRows(t *testing.T) {
	csvData := strings.Join([]string{
		"service_type,storage_class,technology,replication_factor,power_coefficient_wh_per_tbh",
		"ebs,GP3,SSD,2,1.2",
		"short",
		",STANDARD,SSD,3,1.2",
		"s3,,SSD,3,1.2",
		"ebs,BADREPL,SSD,0,1.2",
		"ebs,NEGPOWER,SSD,2,-1",
		"s3,STANDARD,HDD,3,0.65",
	}, "\n")

	logs := captureCarbonLogs(t)
	swapStorageSpecs(t, csvData)

	ebs, ok := GetStorageSpec("ebs", "gp3")
	require.True(t, ok)
	assert.Equal(t, "SSD", ebs.Technology)
	assert.Equal(t, 2, ebs.ReplicationFactor)
	assert.InDelta(t, 1.2, ebs.PowerCoefficient, 1e-9)

	s3, ok := GetStorageSpec("s3", "STANDARD")
	require.True(t, ok)
	assert.Equal(t, 3, s3.ReplicationFactor)
	assert.InDelta(t, 0.65, s3.PowerCoefficient, 1e-9)

	for _, class := range []string{"BADREPL", "NEGPOWER"} {
		_, found := GetStorageSpec("ebs", class)
		assert.False(t, found, "class %s should be skipped", class)
	}
	_, found := GetStorageSpec("s3", "")
	assert.False(t, found)
	_, found = GetStorageSpec("", "STANDARD")
	assert.False(t, found)
	assert.Contains(t, logs.String(), "skipping malformed storage specs CSV row")
}

func captureCarbonLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prevLogger := logger
	prevLevel := zerolog.GlobalLevel()
	logger = zerolog.New(&buf).Level(zerolog.WarnLevel)
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
	t.Cleanup(func() {
		logger = prevLogger
		zerolog.SetGlobalLevel(prevLevel)
	})
	return &buf
}

func swapInstanceSpecs(t *testing.T, data string) {
	t.Helper()
	prevCSV := instanceSpecsCSV
	prevOnce := instanceSpecsOnce
	prevMap := instanceSpecs
	instanceSpecsCSV = data
	instanceSpecsOnce = sync.Once{}
	instanceSpecs = nil
	t.Cleanup(func() {
		instanceSpecsCSV = prevCSV
		instanceSpecsOnce = prevOnce
		instanceSpecs = prevMap
	})
}

func swapGPUSpecs(t *testing.T, data string) {
	t.Helper()
	prevCSV := gpuSpecsCSV
	prevOnce := gpuSpecsOnce
	prevMap := gpuSpecs
	gpuSpecsCSV = data
	gpuSpecsOnce = sync.Once{}
	gpuSpecs = nil
	t.Cleanup(func() {
		gpuSpecsCSV = prevCSV
		gpuSpecsOnce = prevOnce
		gpuSpecs = prevMap
	})
}

func swapStorageSpecs(t *testing.T, data string) {
	t.Helper()
	prevCSV := storageSpecsCSV
	prevOnce := storageSpecsOnce
	prevMap := storageSpecs
	storageSpecsCSV = data
	storageSpecsOnce = sync.Once{}
	storageSpecs = nil
	t.Cleanup(func() {
		storageSpecsCSV = prevCSV
		storageSpecsOnce = prevOnce
		storageSpecs = prevMap
	})
}

func instanceSpecRow(instanceType, vcpu, minWatts, maxWatts string) string {
	cols := repeat("x", colMaxWatts+1)
	cols[colInstanceType] = instanceType
	cols[colVCPUCount] = vcpu
	cols[colMinWatts] = minWatts
	cols[colMaxWatts] = maxWatts
	for i, col := range cols {
		cols[i] = csvEscape(col)
	}
	return strings.Join(cols, ",")
}

func csvEscape(value string) string {
	if strings.ContainsAny(value, ",\"\n") {
		return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
	}
	return value
}

func repeat(value string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = value
	}
	return out
}
