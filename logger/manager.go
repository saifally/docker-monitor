package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type LogManager struct {
	drivers []LogDriver
	configs []LogConfig
}

func NewLogManager() *LogManager {
	return &LogManager{
		drivers: make([]LogDriver, 0),
		configs: make([]LogConfig, 0),
	}
}

func (lm *LogManager) LoadConfigFromEnv() error {
	// Get logging configuration from environment variables
	configStr := os.Getenv("LOG_CONFIG")
	if configStr == "" {
		// Default to console logging
		lm.AddDriver(LogConfig{
			Driver: DriverConsole,
			Options: map[string]string{
				"format": "json",
				"level":  "info",
			},
		})
		return nil
	}

	// Parse JSON configuration
	var configs []LogConfig
	if err := json.Unmarshal([]byte(configStr), &configs); err != nil {
		return fmt.Errorf("failed to parse LOG_CONFIG: %w", err)
	}

	// Add each configured driver
	for _, config := range configs {
		if err := lm.AddDriver(config); err != nil {
			return fmt.Errorf("failed to add driver %s: %w", config.Driver, err)
		}
	}

	return nil
}

func (lm *LogManager) LoadConfigFromFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var configs []LogConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Add each configured driver
	for _, config := range configs {
		if err := lm.AddDriver(config); err != nil {
			return fmt.Errorf("failed to add driver %s: %w", config.Driver, err)
		}
	}

	return nil
}

func (lm *LogManager) AddDriver(config LogConfig) error {
	var driver LogDriver

	// Initialize options map if nil
	if config.Options == nil {
		config.Options = make(map[string]string)
	}

	// Create driver based on type
	switch strings.ToLower(config.Driver) {
	case DriverConsole:
		driver = NewConsoleDriver(config)
	case DriverCSV:
		driver = NewCSVDriver(config)
	case DriverExcel:
		driver = NewExcelDriver(config)
	case DriverCloudWatch:
		driver = NewCloudWatchDriver(config)
	default:
		return fmt.Errorf("unsupported driver: %s", config.Driver)
	}

	// Initialize driver
	if err := driver.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize driver %s: %w", config.Driver, err)
	}

	lm.drivers = append(lm.drivers, driver)
	lm.configs = append(lm.configs, config)

	return nil
}

func (lm *LogManager) LogMetrics(metrics []ContainerMetrics) error {
	var lastErr error

	for i, driver := range lm.drivers {
		if err := driver.LogMetrics(metrics); err != nil {
			lastErr = err
			// Log error to console if this isn't the console driver
			if lm.configs[i].Driver != DriverConsole {
				fmt.Printf("Error logging to %s driver: %v\n", lm.configs[i].Driver, err)
			}
		}
	}

	return lastErr
}

func (lm *LogManager) LogInfo(message string, fields map[string]interface{}) error {
	var lastErr error

	for _, driver := range lm.drivers {
		if err := driver.LogInfo(message, fields); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

func (lm *LogManager) LogError(message string, err error) error {
	var lastErr error

	for _, driver := range lm.drivers {
		if driverErr := driver.LogError(message, err); driverErr != nil {
			lastErr = driverErr
		}
	}

	return lastErr
}

func (lm *LogManager) Close() error {
	var lastErr error

	for _, driver := range lm.drivers {
		if err := driver.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

func (lm *LogManager) GetDriverCount() int {
	return len(lm.drivers)
}

func (lm *LogManager) GetDriverTypes() []string {
	types := make([]string, len(lm.configs))
	for i, config := range lm.configs {
		types[i] = config.Driver
	}
	return types
}

// CreateSampleConfig creates a sample configuration file
func CreateSampleConfig(filePath string) error {
	sampleConfigs := []LogConfig{
		{
			Driver:     DriverConsole,
			OutputPath: "",
			Options: map[string]string{
				"format": "json",
				"level":  "info",
			},
		},
		{
			Driver:     DriverCSV,
			OutputPath: "/logs/container_metrics.csv",
			Options: map[string]string{
				"info_log_file":  "/logs/info.log",
				"error_log_file": "/logs/errors.log",
			},
		},
		{
			Driver:     DriverExcel,
			OutputPath: "/logs/container_metrics.xlsx",
			Options: map[string]string{
				"sheet_name": "Container Metrics",
			},
		},
		{
			Driver:     DriverCloudWatch,
			OutputPath: "",
			Options: map[string]string{
				"log_group":      "/docker-monitor/container-metrics",
				"log_stream":     "container-monitor",
				"region":         "us-east-1",
				"retention_days": "30",
			},
		},
	}

	data, err := json.MarshalIndent(sampleConfigs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}