package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

type ConsoleDriver struct {
	logger *logrus.Logger
	config LogConfig
}

func NewConsoleDriver(config LogConfig) *ConsoleDriver {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	// Set format based on options
	if format, exists := config.Options["format"]; exists && format == "text" {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	} else {
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	// Set log level
	if level, exists := config.Options["level"]; exists {
		if parsedLevel, err := logrus.ParseLevel(level); err == nil {
			logger.SetLevel(parsedLevel)
		}
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	return &ConsoleDriver{
		logger: logger,
		config: config,
	}
}

func (c *ConsoleDriver) Initialize() error {
	c.logger.Info("Console logging driver initialized")
	return nil
}

func (c *ConsoleDriver) LogMetrics(metrics []ContainerMetrics) error {
	// Log summary
	c.logger.WithFields(logrus.Fields{
		"containers_count": len(metrics),
		"timestamp":       time.Now(),
	}).Info("Container monitoring report")

	// Log individual metrics
	for _, metric := range metrics {
		fields := logrus.Fields{
			"container_name": metric.Name,
			"status":        metric.Status,
			"image":         metric.Image,
			"cpu_percent":   metric.CPUPercent,
			"memory_usage":  metric.MemoryUsage,
			"memory_limit":  metric.MemoryLimit,
			"project":       metric.Project,
			"timestamp":     metric.Timestamp,
		}

		// Add compose service if available
		if service, exists := metric.Labels["com.docker.compose.service"]; exists {
			fields["compose_service"] = service
		}

		c.logger.WithFields(fields).Info("Container metrics")
	}

	return nil
}

func (c *ConsoleDriver) LogInfo(message string, fields map[string]interface{}) error {
	c.logger.WithFields(fields).Info(message)
	return nil
}

func (c *ConsoleDriver) LogError(message string, err error) error {
	c.logger.WithError(err).Error(message)
	return nil
}

func (c *ConsoleDriver) Close() error {
	c.logger.Info("Console logging driver closed")
	return nil
}

// Helper method for pretty printing when in text mode
func (c *ConsoleDriver) prettyPrint(metrics []ContainerMetrics) {
	fmt.Printf("\n=== Container Monitoring Report ===\n")
	fmt.Printf("Timestamp: %s\n", time.Now().Format(time.RFC3339))
	fmt.Printf("Containers: %d\n\n", len(metrics))

	for i, metric := range metrics {
		fmt.Printf("Container %d:\n", i+1)
		fmt.Printf("  Name: %s\n", metric.Name)
		fmt.Printf("  Status: %s\n", metric.Status)
		fmt.Printf("  Image: %s\n", metric.Image)
		fmt.Printf("  CPU: %.2f%%\n", metric.CPUPercent)
		fmt.Printf("  Memory: %s / %s\n", metric.MemoryUsage, metric.MemoryLimit)
		fmt.Printf("  Project: %s\n", metric.Project)
		if service, exists := metric.Labels["com.docker.compose.service"]; exists {
			fmt.Printf("  Service: %s\n", service)
		}
		fmt.Println()
	}
}