package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"docker-monitor/logger"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)


type Monitor struct {
	client      *client.Client
	logManager  *logger.LogManager
	frequency   time.Duration
	project     string
}

func NewMonitor() (*Monitor, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Initialize log manager
	logManager := logger.NewLogManager()

	// Try to load config from file first, then environment
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = "/app/config.json"
	}

	if _, err := os.Stat(configFile); err == nil {
		if err := logManager.LoadConfigFromFile(configFile); err != nil {
			return nil, fmt.Errorf("failed to load config from file %s: %w", configFile, err)
		}
	} else {
		if err := logManager.LoadConfigFromEnv(); err != nil {
			return nil, fmt.Errorf("failed to load log config: %w", err)
		}
	}

	// Get monitoring frequency from environment variable (default: 30 seconds)
	frequencyStr := os.Getenv("MONITOR_FREQUENCY")
	frequency := 30 * time.Second
	if frequencyStr != "" {
		if f, err := time.ParseDuration(frequencyStr); err == nil {
			frequency = f
		}
	}

	// Get Docker Compose project name from environment
	project := os.Getenv("COMPOSE_PROJECT_NAME")
	if project == "" {
		// Try alternative environment variables
		project = os.Getenv("DOCKER_COMPOSE_PROJECT")
		if project == "" {
			project = "default"
		}
	}

	return &Monitor{
		client:      cli,
		logManager:  logManager,
		frequency:   frequency,
		project:     project,
	}, nil
}

func (m *Monitor) getContainerStats(ctx context.Context, containerID string) (*types.StatsJSON, error) {
	stats, err := m.client.ContainerStats(ctx, containerID, false)
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

func (m *Monitor) calculateCPUPercent(stats *types.StatsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage) - float64(stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage) - float64(stats.PreCPUStats.SystemUsage)

	if systemDelta > 0 && cpuDelta > 0 {
		return (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return 0.0
}

func (m *Monitor) formatBytes(bytes uint64) string {
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

func (m *Monitor) isFromSameComposeProject(labels map[string]string) bool {
	// Check various Docker Compose label patterns
	composeLabels := []string{
		"com.docker.compose.project",
		"com.docker.compose.project.name",
		"docker-compose.project",
	}

	for _, label := range composeLabels {
		if project, exists := labels[label]; exists {
			return project == m.project
		}
	}

	// Also check if the container name starts with the project name
	if containerName, exists := labels["com.docker.compose.service"]; exists {
		return strings.HasPrefix(containerName, m.project)
	}

	return false
}

func (m *Monitor) monitorContainers(ctx context.Context) error {
	containers, err := m.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	var metrics []logger.ContainerMetrics

	for _, container := range containers {
		// Skip if not from the same Docker Compose project
		if !m.isFromSameComposeProject(container.Labels) {
			continue
		}

		metric := logger.ContainerMetrics{
			Name:      strings.TrimPrefix(container.Names[0], "/"),
			Status:    container.Status,
			Image:     container.Image,
			Timestamp: time.Now(),
			Labels:    container.Labels,
			Project:   m.project,
		}

		// Get container stats only if running
		if container.State == "running" {
			stats, err := m.getContainerStats(ctx, container.ID)
			if err != nil {
				m.logManager.LogError(fmt.Sprintf("Failed to get stats for container %s", metric.Name), err)
			} else {
				metric.CPUPercent = m.calculateCPUPercent(stats)
				metric.MemoryUsage = m.formatBytes(stats.MemoryStats.Usage)
				metric.MemoryLimit = m.formatBytes(stats.MemoryStats.Limit)
			}
		}

		metrics = append(metrics, metric)
	}

	// Log metrics using the log manager
	return m.logManager.LogMetrics(metrics)
}

func (m *Monitor) Start(ctx context.Context) error {
	m.logManager.LogInfo("Starting Docker container monitor", map[string]interface{}{
		"project":   m.project,
		"frequency": m.frequency.String(),
	})

	ticker := time.NewTicker(m.frequency)
	defer ticker.Stop()

	// Initial monitoring
	if err := m.monitorContainers(ctx); err != nil {
		m.logManager.LogError("Initial monitoring failed", err)
		return err
	}

	for {
		select {
		case <-ctx.Done():
			m.logManager.LogInfo("Monitor stopping due to context cancellation", nil)
			m.logManager.Close()
			return ctx.Err()
		case <-ticker.C:
			if err := m.monitorContainers(ctx); err != nil {
				m.logManager.LogError("Monitoring cycle failed", err)
			}
		}
	}
}

func main() {
	monitor, err := NewMonitor()
	if err != nil {
		fmt.Printf("Failed to create monitor: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	if err := monitor.Start(ctx); err != nil {
		monitor.logManager.LogError("Monitor failed", err)
		monitor.logManager.Close()
		os.Exit(1)
	}
}