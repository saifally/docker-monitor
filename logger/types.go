package logger

import (
	"time"
)

type ContainerMetrics struct {
	Name        string            `json:"name" csv:"name" excel:"Name"`
	Status      string            `json:"status" csv:"status" excel:"Status"`
	Image       string            `json:"image" csv:"image" excel:"Image"`
	CPUPercent  float64           `json:"cpu_percent" csv:"cpu_percent" excel:"CPU %"`
	MemoryUsage string            `json:"memory_usage" csv:"memory_usage" excel:"Memory Usage"`
	MemoryLimit string            `json:"memory_limit" csv:"memory_limit" excel:"Memory Limit"`
	Timestamp   time.Time         `json:"timestamp" csv:"timestamp" excel:"Timestamp"`
	Labels      map[string]string `json:"labels" csv:"-" excel:"-"`
	Project     string            `json:"project" csv:"project" excel:"Project"`
}

type LogDriver interface {
	Initialize() error
	LogMetrics(metrics []ContainerMetrics) error
	LogInfo(message string, fields map[string]interface{}) error
	LogError(message string, err error) error
	Close() error
}

type LogConfig struct {
	Driver     string            `json:"driver"`
	OutputPath string            `json:"output_path"`
	Options    map[string]string `json:"options"`
}

const (
	DriverConsole    = "console"
	DriverCSV        = "csv"
	DriverExcel      = "excel"
	DriverCloudWatch = "cloudwatch"
)