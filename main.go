package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

type ContainerMetrics struct {
	Name        string            `json:"name"`
	Status      string            `json:"status"`
	Image       string            `json:"image"`
	CPUPercent  float64           `json:"cpu_percent"`
	MemoryUsage string            `json:"memory_usage"`
	MemoryLimit string            `json:"memory_limit"`
	Timestamp   time.Time         `json:"timestamp"`
	Labels      map[string]string `json:"labels"`
}

type Monitor struct {
	client    *client.Client
	logger    *logrus.Logger
	frequency time.Duration
	project   string
}

func NewMonitor() (*Monitor, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

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
		client:    cli,
		logger:    logger,
		frequency: frequency,
		project:   project,
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
	containers, err := m.client.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	var metrics []ContainerMetrics

	for _, container := range containers {
		// Skip if not from the same Docker Compose project
		if !m.isFromSameComposeProject(container.Labels) {
			continue
		}

		metric := ContainerMetrics{
			Name:      strings.TrimPrefix(container.Names[0], "/"),
			Status:    container.Status,
			Image:     container.Image,
			Timestamp: time.Now(),
			Labels:    container.Labels,
		}

		// Get container stats only if running
		if container.State == "running" {
			stats, err := m.getContainerStats(ctx, container.ID)
			if err != nil {
				m.logger.WithError(err).Warnf("Failed to get stats for container %s", metric.Name)
			} else {
				metric.CPUPercent = m.calculateCPUPercent(stats)
				metric.MemoryUsage = m.formatBytes(stats.MemoryStats.Usage)
				metric.MemoryLimit = m.formatBytes(stats.MemoryStats.Limit)
			}
		}

		metrics = append(metrics, metric)
	}

	// Log metrics
	m.logger.WithFields(logrus.Fields{
		"project":           m.project,
		"containers_count":  len(metrics),
		"monitoring_frequency": m.frequency.String(),
	}).Info("Container monitoring report")

	for _, metric := range metrics {
		m.logger.WithFields(logrus.Fields{
			"container_name":   metric.Name,
			"status":          metric.Status,
			"image":           metric.Image,
			"cpu_percent":     metric.CPUPercent,
			"memory_usage":    metric.MemoryUsage,
			"memory_limit":    metric.MemoryLimit,
			"compose_service": metric.Labels["com.docker.compose.service"],
		}).Info("Container metrics")
	}

	return nil
}

func (m *Monitor) Start(ctx context.Context) error {
	m.logger.WithFields(logrus.Fields{
		"project":   m.project,
		"frequency": m.frequency.String(),
	}).Info("Starting Docker container monitor")

	ticker := time.NewTicker(m.frequency)
	defer ticker.Stop()

	// Initial monitoring
	if err := m.monitorContainers(ctx); err != nil {
		m.logger.WithError(err).Error("Initial monitoring failed")
		return err
	}

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Monitor stopping due to context cancellation")
			return ctx.Err()
		case <-ticker.C:
			if err := m.monitorContainers(ctx); err != nil {
				m.logger.WithError(err).Error("Monitoring cycle failed")
			}
		}
	}
}

func main() {
	monitor, err := NewMonitor()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create monitor")
	}

	ctx := context.Background()
	if err := monitor.Start(ctx); err != nil {
		logrus.WithError(err).Fatal("Monitor failed")
	}
}