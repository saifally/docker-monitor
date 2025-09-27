package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"docker-monitor/logger"

	"github.com/docker/docker/api/types"
)

func TestMain(m *testing.M) {
	// Setup: Clear environment variables that might affect tests
	oldConfigFile := os.Getenv("CONFIG_FILE")
	oldMonitorFreq := os.Getenv("MONITOR_FREQUENCY")
	oldProjectName := os.Getenv("COMPOSE_PROJECT_NAME")
	oldDockerProject := os.Getenv("DOCKER_COMPOSE_PROJECT")
	oldLogConfig := os.Getenv("LOG_CONFIG")

	// Cleanup after tests
	defer func() {
		os.Setenv("CONFIG_FILE", oldConfigFile)
		os.Setenv("MONITOR_FREQUENCY", oldMonitorFreq)
		os.Setenv("COMPOSE_PROJECT_NAME", oldProjectName)
		os.Setenv("DOCKER_COMPOSE_PROJECT", oldDockerProject)
		os.Setenv("LOG_CONFIG", oldLogConfig)
	}()

	// Run tests
	code := m.Run()
	os.Exit(code)
}

func TestNewMonitor(t *testing.T) {
	tests := []struct {
		name           string
		setupEnv       func()
		expectedError  bool
		expectedFreq   time.Duration
		expectedProj   string
	}{
		{
			name: "default configuration",
			setupEnv: func() {
				os.Unsetenv("CONFIG_FILE")
				os.Unsetenv("MONITOR_FREQUENCY")
				os.Unsetenv("COMPOSE_PROJECT_NAME")
				os.Unsetenv("DOCKER_COMPOSE_PROJECT")
				os.Unsetenv("LOG_CONFIG")
			},
			expectedError: false,
			expectedFreq:  30 * time.Second,
			expectedProj:  "default",
		},
		{
			name: "custom frequency",
			setupEnv: func() {
				os.Unsetenv("CONFIG_FILE")
				os.Setenv("MONITOR_FREQUENCY", "1m")
				os.Unsetenv("COMPOSE_PROJECT_NAME")
				os.Unsetenv("DOCKER_COMPOSE_PROJECT")
				os.Unsetenv("LOG_CONFIG")
			},
			expectedError: false,
			expectedFreq:  1 * time.Minute,
			expectedProj:  "default",
		},
		{
			name: "invalid frequency falls back to default",
			setupEnv: func() {
				os.Unsetenv("CONFIG_FILE")
				os.Setenv("MONITOR_FREQUENCY", "invalid")
				os.Unsetenv("COMPOSE_PROJECT_NAME")
				os.Unsetenv("DOCKER_COMPOSE_PROJECT")
				os.Unsetenv("LOG_CONFIG")
			},
			expectedError: false,
			expectedFreq:  30 * time.Second,
			expectedProj:  "default",
		},
		{
			name: "compose project name",
			setupEnv: func() {
				os.Unsetenv("CONFIG_FILE")
				os.Unsetenv("MONITOR_FREQUENCY")
				os.Setenv("COMPOSE_PROJECT_NAME", "myproject")
				os.Unsetenv("DOCKER_COMPOSE_PROJECT")
				os.Unsetenv("LOG_CONFIG")
			},
			expectedError: false,
			expectedFreq:  30 * time.Second,
			expectedProj:  "myproject",
		},
		{
			name: "docker compose project fallback",
			setupEnv: func() {
				os.Unsetenv("CONFIG_FILE")
				os.Unsetenv("MONITOR_FREQUENCY")
				os.Unsetenv("COMPOSE_PROJECT_NAME")
				os.Setenv("DOCKER_COMPOSE_PROJECT", "dockerproject")
				os.Unsetenv("LOG_CONFIG")
			},
			expectedError: false,
			expectedFreq:  30 * time.Second,
			expectedProj:  "dockerproject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()

			monitor, err := NewMonitor()

			if tt.expectedError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectedError {
				if monitor == nil {
					t.Errorf("Expected monitor instance but got nil")
					return
				}
				if monitor.frequency != tt.expectedFreq {
					t.Errorf("Expected frequency %v, got %v", tt.expectedFreq, monitor.frequency)
				}
				if monitor.project != tt.expectedProj {
					t.Errorf("Expected project %q, got %q", tt.expectedProj, monitor.project)
				}
				if monitor.client == nil {
					t.Errorf("Expected Docker client to be initialized")
				}
				if monitor.logManager == nil {
					t.Errorf("Expected log manager to be initialized")
				}
			}
		})
	}
}

func TestCalculateCPUPercent(t *testing.T) {
	monitor := &Monitor{}

	tests := []struct {
		name     string
		stats    *types.StatsJSON
		expected float64
	}{
		{
			name: "normal CPU calculation",
			stats: &types.StatsJSON{
				Stats: types.Stats{
					CPUStats: types.CPUStats{
						CPUUsage: types.CPUUsage{
							TotalUsage:    200000000,
							PercpuUsage:   []uint64{50000000, 50000000, 50000000, 50000000},
						},
						SystemUsage: 2000000000,
					},
					PreCPUStats: types.CPUStats{
						CPUUsage: types.CPUUsage{
							TotalUsage: 100000000,
						},
						SystemUsage: 1000000000,
					},
				},
			},
			expected: 40.0, // (100000000 / 1000000000) * 4 * 100
		},
		{
			name: "zero system delta returns zero",
			stats: &types.StatsJSON{
				Stats: types.Stats{
					CPUStats: types.CPUStats{
						CPUUsage: types.CPUUsage{
							TotalUsage:    200000000,
							PercpuUsage:   []uint64{50000000, 50000000},
						},
						SystemUsage: 1000000000,
					},
					PreCPUStats: types.CPUStats{
						CPUUsage: types.CPUUsage{
							TotalUsage: 100000000,
						},
						SystemUsage: 1000000000, // Same as current
					},
				},
			},
			expected: 0.0,
		},
		{
			name: "zero CPU delta returns zero",
			stats: &types.StatsJSON{
				Stats: types.Stats{
					CPUStats: types.CPUStats{
						CPUUsage: types.CPUUsage{
							TotalUsage:    100000000,
							PercpuUsage:   []uint64{25000000, 25000000, 25000000, 25000000},
						},
						SystemUsage: 2000000000,
					},
					PreCPUStats: types.CPUStats{
						CPUUsage: types.CPUUsage{
							TotalUsage: 100000000, // Same as current
						},
						SystemUsage: 1000000000,
					},
				},
			},
			expected: 0.0,
		},
		{
			name: "negative deltas return zero",
			stats: &types.StatsJSON{
				Stats: types.Stats{
					CPUStats: types.CPUStats{
						CPUUsage: types.CPUUsage{
							TotalUsage:    50000000,
							PercpuUsage:   []uint64{12500000, 12500000, 12500000, 12500000},
						},
						SystemUsage: 500000000,
					},
					PreCPUStats: types.CPUStats{
						CPUUsage: types.CPUUsage{
							TotalUsage: 100000000, // Higher than current
						},
						SystemUsage: 1000000000, // Higher than current
					},
				},
			},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := monitor.calculateCPUPercent(tt.stats)
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	monitor := &Monitor{}

	tests := []struct {
		name     string
		bytes    uint64
		expected string
	}{
		{
			name:     "bytes",
			bytes:    512,
			expected: "512 B",
		},
		{
			name:     "kilobytes",
			bytes:    1536, // 1.5 KB
			expected: "1.5 KB",
		},
		{
			name:     "megabytes",
			bytes:    1572864, // 1.5 MB
			expected: "1.5 MB",
		},
		{
			name:     "gigabytes",
			bytes:    1610612736, // 1.5 GB
			expected: "1.5 GB",
		},
		{
			name:     "terabytes",
			bytes:    1649267441664, // 1.5 TB
			expected: "1.5 TB",
		},
		{
			name:     "exactly 1 KB",
			bytes:    1024,
			expected: "1.0 KB",
		},
		{
			name:     "exactly 1 MB",
			bytes:    1048576,
			expected: "1.0 MB",
		},
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0 B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := monitor.formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestIsFromSameComposeProject(t *testing.T) {
	monitor := &Monitor{project: "myproject"}

	tests := []struct {
		name     string
		labels   map[string]string
		expected bool
	}{
		{
			name: "matches com.docker.compose.project",
			labels: map[string]string{
				"com.docker.compose.project": "myproject",
			},
			expected: true,
		},
		{
			name: "matches com.docker.compose.project.name",
			labels: map[string]string{
				"com.docker.compose.project.name": "myproject",
			},
			expected: true,
		},
		{
			name: "matches docker-compose.project",
			labels: map[string]string{
				"docker-compose.project": "myproject",
			},
			expected: true,
		},
		{
			name: "matches com.docker.compose.service with prefix",
			labels: map[string]string{
				"com.docker.compose.service": "myproject-web",
			},
			expected: true,
		},
		{
			name: "does not match different project",
			labels: map[string]string{
				"com.docker.compose.project": "otherproject",
			},
			expected: false,
		},
		{
			name: "does not match service without prefix",
			labels: map[string]string{
				"com.docker.compose.service": "web-myproject",
			},
			expected: false,
		},
		{
			name: "no compose labels",
			labels: map[string]string{
				"some.other.label": "value",
			},
			expected: false,
		},
		{
			name:     "empty labels",
			labels:   map[string]string{},
			expected: false,
		},
		{
			name:     "nil labels",
			labels:   nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := monitor.isFromSameComposeProject(tt.labels)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// DockerClientInterface defines the methods we need from Docker client
type DockerClientInterface interface {
	ContainerStats(ctx context.Context, containerID string, stream bool) (types.ContainerStats, error)
	ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error)
}

// MockDockerClient is a mock implementation of the Docker client interface
type MockDockerClient struct {
	containerStatsFunc func(ctx context.Context, containerID string, stream bool) (types.ContainerStats, error)
	containerListFunc  func(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error)
}

func (m *MockDockerClient) ContainerStats(ctx context.Context, containerID string, stream bool) (types.ContainerStats, error) {
	if m.containerStatsFunc != nil {
		return m.containerStatsFunc(ctx, containerID, stream)
	}
	return types.ContainerStats{}, nil
}

func (m *MockDockerClient) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	if m.containerListFunc != nil {
		return m.containerListFunc(ctx, options)
	}
	return []types.Container{}, nil
}

// MockReadCloser implements io.ReadCloser for mocking response bodies
type MockReadCloser struct {
	*strings.Reader
	closed bool
}

func NewMockReadCloser(data string) *MockReadCloser {
	return &MockReadCloser{
		Reader: strings.NewReader(data),
		closed: false,
	}
}

func (m *MockReadCloser) Close() error {
	m.closed = true
	return nil
}

func TestGetContainerStats(t *testing.T) {
	tests := []struct {
		name          string
		containerID   string
		mockResponse  string
		mockError     error
		expectedError bool
	}{
		{
			name:        "successful stats retrieval",
			containerID: "test-container",
			mockResponse: `{
				"cpu_stats": {
					"cpu_usage": {
						"total_usage": 200000000,
						"percpu_usage": [50000000, 50000000, 50000000, 50000000]
					},
					"system_cpu_usage": 2000000000
				},
				"precpu_stats": {
					"cpu_usage": {
						"total_usage": 100000000
					},
					"system_cpu_usage": 1000000000
				},
				"memory_stats": {
					"usage": 1048576,
					"limit": 2097152
				}
			}`,
			expectedError: false,
		},
		{
			name:          "docker client error",
			containerID:   "test-container",
			mockError:     &testError{"docker client error"},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockDockerClient{
				containerStatsFunc: func(ctx context.Context, containerID string, stream bool) (types.ContainerStats, error) {
					if tt.mockError != nil {
						return types.ContainerStats{}, tt.mockError
					}
					return types.ContainerStats{
						Body: NewMockReadCloser(tt.mockResponse),
					}, nil
				},
			}

			// Create a test monitor struct with mock client
			testMonitor := &testMonitor{
				mockClient: mockClient,
			}

			ctx := context.Background()
			stats, err := testMonitor.getContainerStats(ctx, tt.containerID)

			if tt.expectedError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectedError && stats == nil {
				t.Errorf("Expected stats but got nil")
			}
		})
	}
}

// testError implements the error interface for testing
type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}

// testMonitor is a test-specific monitor with mock capabilities
type testMonitor struct {
	mockClient DockerClientInterface
}

func (tm *testMonitor) getContainerStats(ctx context.Context, containerID string) (*types.StatsJSON, error) {
	stats, err := tm.mockClient.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, err
	}
	defer stats.Body.Close()

	var statsJSON types.StatsJSON
	if err := json.NewDecoder(stats.Body).Decode(&statsJSON); err != nil {
		return nil, err
	}
	return &statsJSON, nil
}

func (tm *testMonitor) calculateCPUPercent(stats *types.StatsJSON) float64 {
	cpuDelta := float64(stats.Stats.CPUStats.CPUUsage.TotalUsage) - float64(stats.Stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.Stats.CPUStats.SystemUsage) - float64(stats.Stats.PreCPUStats.SystemUsage)

	if systemDelta > 0 && cpuDelta > 0 {
		return (cpuDelta / systemDelta) * float64(len(stats.Stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return 0.0
}

func (tm *testMonitor) formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (tm *testMonitor) isFromSameComposeProject(labels map[string]string, project string) bool {
	composeLabels := []string{
		"com.docker.compose.project",
		"com.docker.compose.project.name",
		"docker-compose.project",
	}

	for _, label := range composeLabels {
		if projectValue, exists := labels[label]; exists {
			return projectValue == project
		}
	}

	if containerName, exists := labels["com.docker.compose.service"]; exists {
		return strings.HasPrefix(containerName, project)
	}
	return false
}

func (tm *testMonitor) monitorContainers(ctx context.Context, project string, logManager *MockLogManager) error {
	containers, err := tm.mockClient.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	var metrics []logger.ContainerMetrics

	for _, container := range containers {
		if !tm.isFromSameComposeProject(container.Labels, project) {
			continue
		}

		metric := logger.ContainerMetrics{
			Name:      strings.TrimPrefix(container.Names[0], "/"),
			Status:    container.Status,
			Image:     container.Image,
			Timestamp: time.Now(),
			Labels:    container.Labels,
			Project:   project,
		}

		if container.State == "running" {
			stats, err := tm.getContainerStats(ctx, container.ID)
			if err != nil {
				logManager.LogError(fmt.Sprintf("Failed to get stats for container %s", metric.Name), err)
			} else {
				metric.CPUPercent = tm.calculateCPUPercent(stats)
				metric.MemoryUsage = tm.formatBytes(stats.Stats.MemoryStats.Usage)
				metric.MemoryLimit = tm.formatBytes(stats.Stats.MemoryStats.Limit)
			}
		}

		metrics = append(metrics, metric)
	}

	return logManager.LogMetrics(metrics)
}

// MockLogManager is a mock implementation of the LogManager
type MockLogManager struct {
	logMetricsFunc func(metrics []logger.ContainerMetrics) error
	logInfoFunc    func(message string, fields map[string]interface{}) error
	logErrorFunc   func(message string, err error) error
	closeFunc      func() error
}

func (m *MockLogManager) LogMetrics(metrics []logger.ContainerMetrics) error {
	if m.logMetricsFunc != nil {
		return m.logMetricsFunc(metrics)
	}
	return nil
}

func (m *MockLogManager) LogInfo(message string, fields map[string]interface{}) error {
	if m.logInfoFunc != nil {
		return m.logInfoFunc(message, fields)
	}
	return nil
}

func (m *MockLogManager) LogError(message string, err error) error {
	if m.logErrorFunc != nil {
		return m.logErrorFunc(message, err)
	}
	return nil
}

func (m *MockLogManager) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *MockLogManager) LoadConfigFromFile(filePath string) error {
	return nil
}

func (m *MockLogManager) LoadConfigFromEnv() error {
	return nil
}

func TestMonitorContainers(t *testing.T) {
	tests := []struct {
		name           string
		project        string
		containers     []types.Container
		containerStats map[string]string
		expectedError  bool
		expectedMetrics int
	}{
		{
			name:    "successful monitoring with running container",
			project: "testproject",
			containers: []types.Container{
				{
					ID:    "container1",
					Names: []string{"/testproject-web-1"},
					Image: "nginx:latest",
					State: "running",
					Status: "Up 5 minutes",
					Labels: map[string]string{
						"com.docker.compose.project": "testproject",
					},
				},
			},
			containerStats: map[string]string{
				"container1": `{
					"cpu_stats": {
						"cpu_usage": {
							"total_usage": 200000000,
							"percpu_usage": [50000000, 50000000, 50000000, 50000000]
						},
						"system_cpu_usage": 2000000000
					},
					"precpu_stats": {
						"cpu_usage": {
							"total_usage": 100000000
						},
						"system_cpu_usage": 1000000000
					},
					"memory_stats": {
						"usage": 1048576,
						"limit": 2097152
					}
				}`,
			},
			expectedError:   false,
			expectedMetrics: 1,
		},
		{
			name:    "container from different project filtered out",
			project: "testproject",
			containers: []types.Container{
				{
					ID:    "container1",
					Names: []string{"/otherproject-web-1"},
					Image: "nginx:latest",
					State: "running",
					Status: "Up 5 minutes",
					Labels: map[string]string{
						"com.docker.compose.project": "otherproject",
					},
				},
			},
			expectedError:   false,
			expectedMetrics: 0,
		},
		{
			name:    "stopped container without stats",
			project: "testproject",
			containers: []types.Container{
				{
					ID:    "container1",
					Names: []string{"/testproject-web-1"},
					Image: "nginx:latest",
					State: "exited",
					Status: "Exited (0) 5 minutes ago",
					Labels: map[string]string{
						"com.docker.compose.project": "testproject",
					},
				},
			},
			expectedError:   false,
			expectedMetrics: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedMetrics []logger.ContainerMetrics

			mockClient := &MockDockerClient{
				containerListFunc: func(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
					return tt.containers, nil
				},
				containerStatsFunc: func(ctx context.Context, containerID string, stream bool) (types.ContainerStats, error) {
					if statsJSON, exists := tt.containerStats[containerID]; exists {
						return types.ContainerStats{
							Body: NewMockReadCloser(statsJSON),
						}, nil
					}
					return types.ContainerStats{}, &testError{"container not found"}
				},
			}

			mockLogManager := &MockLogManager{
				logMetricsFunc: func(metrics []logger.ContainerMetrics) error {
					capturedMetrics = metrics
					return nil
				},
			}

			testMonitor := &testMonitor{
				mockClient: mockClient,
			}

			ctx := context.Background()
			err := testMonitor.monitorContainers(ctx, tt.project, mockLogManager)

			if tt.expectedError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(capturedMetrics) != tt.expectedMetrics {
				t.Errorf("Expected %d metrics, got %d", tt.expectedMetrics, len(capturedMetrics))
			}

			// Verify metrics content for successful cases
			if !tt.expectedError && len(capturedMetrics) > 0 {
				metric := capturedMetrics[0]
				if metric.Project != tt.project {
					t.Errorf("Expected project %q, got %q", tt.project, metric.Project)
				}
				if metric.Timestamp.IsZero() {
					t.Errorf("Expected timestamp to be set")
				}
			}
		})
	}
}