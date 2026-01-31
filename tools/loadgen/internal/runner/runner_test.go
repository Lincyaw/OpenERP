package runner

import (
	"testing"
	"time"

	"github.com/example/erp/tools/loadgen/internal/config"
	"github.com/example/erp/tools/loadgen/internal/loadctrl"
	"github.com/stretchr/testify/require"
)

func TestNew_MinimalConfig(t *testing.T) {
	cfg := &config.Config{
		Name:    "test",
		Version: "1.0",
		Target: config.TargetConfig{
			BaseURL: "http://localhost:8080",
			Timeout: 30 * time.Second,
		},
		Duration: 10 * time.Second,
		Endpoints: []config.EndpointConfig{
			{
				Name:   "test",
				Path:   "/health",
				Method: "GET",
				Weight: 1,
			},
		},
		TrafficShaper: loadctrl.ShaperConfig{
			Type:    "step",
			BaseQPS: 10,
		},
	}
	cfg.ApplyDefaults()

	runner, err := New(cfg)
	require.NoError(t, err)
	require.NotNil(t, runner)
	require.NotNil(t, runner.httpClient)
	require.NotNil(t, runner.pool)
	require.NotNil(t, runner.metrics)
	require.NotNil(t, runner.controller)
	require.NotNil(t, runner.workerPool)
}

func TestExtractJSONPath(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		path     string
		multiple bool
		want     []any
	}{
		{
			name:     "simple path",
			body:     `{"data": {"id": "123"}}`,
			path:     "$.data.id",
			multiple: false,
			want:     []any{"123"},
		},
		{
			name:     "array extraction",
			body:     `{"data": [{"id": "1"}, {"id": "2"}]}`,
			path:     "$.data[*].id",
			multiple: true,
			want:     []any{"1", "2"},
		},
		{
			name:     "root array",
			body:     `[{"id": "1"}, {"id": "2"}]`,
			path:     "$[*].id",
			multiple: true,
			want:     []any{"1", "2"},
		},
		{
			name:     "nested object",
			body:     `{"response": {"data": {"user": {"name": "test"}}}}`,
			path:     "$.response.data.user.name",
			multiple: false,
			want:     []any{"test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSONPath([]byte(tt.body), tt.path, tt.multiple)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestTruncate(t *testing.T) {
	require.Equal(t, "hello", truncate("hello", 10))
	require.Equal(t, "hel...", truncate("hello world", 6))
}
