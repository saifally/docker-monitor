package logger

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type CSVDriver struct {
	config     LogConfig
	filePath   string
	fileHandle *os.File
	writer     *csv.Writer
	headerWritten bool
}

func NewCSVDriver(config LogConfig) *CSVDriver {
	filePath := config.OutputPath
	if filePath == "" {
		filePath = "container_metrics.csv"
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if dir != "." {
		os.MkdirAll(dir, 0755)
	}

	return &CSVDriver{
		config:   config,
		filePath: filePath,
	}
}

func (c *CSVDriver) Initialize() error {
	// Check if file exists to determine if we need to write headers
	_, err := os.Stat(c.filePath)
	fileExists := !os.IsNotExist(err)

	// Open file in append mode
	file, err := os.OpenFile(c.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}

	c.fileHandle = file
	c.writer = csv.NewWriter(file)
	c.headerWritten = fileExists

	// Write header if file is new
	if !c.headerWritten {
		header := []string{
			"timestamp",
			"name",
			"status",
			"image",
			"cpu_percent",
			"memory_usage",
			"memory_limit",
			"project",
			"compose_service",
		}
		if err := c.writer.Write(header); err != nil {
			return fmt.Errorf("failed to write CSV header: %w", err)
		}
		c.writer.Flush()
		c.headerWritten = true
	}

	return nil
}

func (c *CSVDriver) LogMetrics(metrics []ContainerMetrics) error {
	for _, metric := range metrics {
		record := []string{
			metric.Timestamp.Format(time.RFC3339),
			metric.Name,
			metric.Status,
			metric.Image,
			strconv.FormatFloat(metric.CPUPercent, 'f', 2, 64),
			metric.MemoryUsage,
			metric.MemoryLimit,
			metric.Project,
			c.getComposeService(metric.Labels),
		}

		if err := c.writer.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	c.writer.Flush()
	return c.writer.Error()
}

func (c *CSVDriver) LogInfo(message string, fields map[string]interface{}) error {
	// For CSV, we'll write info logs to a separate log file if specified
	if logFile, exists := c.config.Options["info_log_file"]; exists {
		return c.writeInfoLog(logFile, message, fields)
	}
	// Otherwise, skip info logs for CSV driver
	return nil
}

func (c *CSVDriver) LogError(message string, err error) error {
	// For CSV, we'll write error logs to a separate log file if specified
	if logFile, exists := c.config.Options["error_log_file"]; exists {
		fields := map[string]interface{}{
			"error": err.Error(),
		}
		return c.writeInfoLog(logFile, message, fields)
	}
	// Otherwise, skip error logs for CSV driver
	return nil
}

func (c *CSVDriver) Close() error {
	if c.writer != nil {
		c.writer.Flush()
	}
	if c.fileHandle != nil {
		return c.fileHandle.Close()
	}
	return nil
}

func (c *CSVDriver) getComposeService(labels map[string]string) string {
	if service, exists := labels["com.docker.compose.service"]; exists {
		return service
	}
	return ""
}

func (c *CSVDriver) writeInfoLog(logFile, message string, fields map[string]interface{}) error {
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	timestamp := time.Now().Format(time.RFC3339)
	logLine := fmt.Sprintf("%s - %s", timestamp, message)

	if len(fields) > 0 {
		logLine += " - Fields: "
		for k, v := range fields {
			logLine += fmt.Sprintf("%s=%v ", k, v)
		}
	}

	logLine += "\n"
	_, err = file.WriteString(logLine)
	return err
}

// RotateFile creates a new CSV file with timestamp suffix
func (c *CSVDriver) RotateFile() error {
	if c.fileHandle != nil {
		c.Close()
	}

	// Create new filename with timestamp
	ext := filepath.Ext(c.filePath)
	base := c.filePath[:len(c.filePath)-len(ext)]
	timestamp := time.Now().Format("20060102_150405")
	newPath := fmt.Sprintf("%s_%s%s", base, timestamp, ext)

	// Rename current file
	if err := os.Rename(c.filePath, newPath); err != nil {
		return err
	}

	// Reset header flag and reinitialize
	c.headerWritten = false
	return c.Initialize()
}