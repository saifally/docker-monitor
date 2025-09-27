// Package main defines the entry point for the Docker container monitoring application.
// In Go, every executable program must have a main package with a main() function.
package main

// Import statement brings in external packages and libraries.
// Go organizes code into packages - collections of related functions and types.
import (
	// Standard library packages (built into Go)
	"context"        // Provides context for managing goroutines and cancellation
	"encoding/json"  // JSON encoding/decoding functionality
	"fmt"            // Formatted I/O functions (like printf in C)
	"os"             // Operating system interface (environment variables, file operations)
	"strings"        // String manipulation functions
	"time"           // Time and duration handling

	// Local package import - references our custom logger package
	// The path is relative to the module root defined in go.mod
	"docker-monitor/logger"

	// Third-party packages (external dependencies)
	"github.com/docker/docker/api/types" // Docker API type definitions
	"github.com/docker/docker/client"    // Docker client library
)

// Monitor is a struct type that encapsulates all the data and behavior
// for monitoring Docker containers. In Go, structs are similar to classes
// in other languages, but without inheritance.
type Monitor struct {
	// Struct fields define the data that each Monitor instance will contain

	// *client.Client is a pointer to a Docker client instance
	// Pointers in Go store memory addresses rather than values
	client *client.Client

	// *logger.LogManager is a pointer to our custom logging manager
	// This handles multiple logging drivers (console, CSV, Excel, CloudWatch)
	logManager *logger.LogManager

	// time.Duration represents a span of time (like 30 seconds)
	// It's an int64 type that counts nanoseconds
	frequency time.Duration

	// string is Go's built-in string type (UTF-8 encoded)
	project string
}

// NewMonitor is a constructor function that creates and initializes a new Monitor instance.
// In Go, constructor functions are a common pattern since there are no class constructors.
// The function returns a pointer to Monitor and an error (Go's error handling pattern).
func NewMonitor() (*Monitor, error) {
	// := is Go's short variable declaration - it declares and assigns in one step
	// The type is inferred from the right-hand side
	cli, err := client.NewClientWithOpts(
		client.FromEnv,                        // Option: use environment variables for Docker connection
		client.WithAPIVersionNegotiation(),   // Option: automatically negotiate API version
	)

	// Go's explicit error handling - always check if err is not nil
	if err != nil {
		// fmt.Errorf creates a formatted error message
		// %w verb wraps the original error (error wrapping pattern)
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Initialize log manager by calling constructor from our logger package
	logManager := logger.NewLogManager()

	// os.Getenv reads environment variables from the operating system
	// It returns an empty string if the variable doesn't exist
	configFile := os.Getenv("CONFIG_FILE")

	// Go's zero value for strings is "" (empty string)
	// This if statement checks if the environment variable was not set
	if configFile == "" {
		// Set a default value for YAML config
		configFile = "/app/config.yaml"
	}

	// os.Stat returns file information and an error
	// We use the blank identifier _ to ignore the file info we don't need
	if _, err := os.Stat(configFile); err == nil {
		// err == nil means the file exists (no error from Stat)
		if err := logManager.LoadConfigFromFile(configFile); err != nil {
			// Return early if configuration loading fails
			return nil, fmt.Errorf("failed to load config from file %s: %w", configFile, err)
		}
	} else {
		// File doesn't exist, try loading from environment variables
		if err := logManager.LoadConfigFromEnv(); err != nil {
			return nil, fmt.Errorf("failed to load log config: %w", err)
		}
	}

	// time.Second is a predefined constant representing one second
	// Multiplication with integers gives us different durations
	frequencyStr := os.Getenv("MONITOR_FREQUENCY")
	frequency := 30 * time.Second // Default: 30 seconds

	// String comparison in Go - comparing if string is not empty
	if frequencyStr != "" {
		// time.ParseDuration parses strings like "30s", "1m", "2h"
		// Multiple assignment - Go functions can return multiple values
		if f, err := time.ParseDuration(frequencyStr); err == nil {
			// Only update frequency if parsing succeeded (err == nil)
			frequency = f
		}
		// If parsing fails, we silently keep the default (defensive programming)
	}

	// Environment variable fallback pattern - try multiple variables
	project := os.Getenv("COMPOSE_PROJECT_NAME")
	if project == "" {
		// Try alternative environment variable
		project = os.Getenv("DOCKER_COMPOSE_PROJECT")
		if project == "" {
			// Final fallback to default value
			project = "default"
		}
	}

	// Return a pointer to a newly created Monitor struct
	// &Monitor{...} creates a struct literal and returns its address
	return &Monitor{
		client:     cli,        // Assign the Docker client
		logManager: logManager, // Assign the log manager
		frequency:  frequency,  // Assign the monitoring frequency
		project:    project,    // Assign the project name
	}, nil // Return nil error indicating success
}

// getContainerStats is a method on the Monitor type (receiver method).
// (m *Monitor) makes this a method - m is the receiver variable.
// ctx context.Context is Go's standard way to handle request contexts.
// It returns a pointer to StatsJSON and an error.
func (m *Monitor) getContainerStats(ctx context.Context, containerID string) (*types.StatsJSON, error) {
	// Call Docker API to get container statistics
	// false parameter means "don't stream" - get a single snapshot
	stats, err := m.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		// Return zero value (nil) and the error
		return nil, err
	}

	// defer schedules a function call to run when the current function returns
	// This ensures resources are cleaned up even if errors occur later
	defer stats.Body.Close()

	// var declares a variable with its zero value
	// Zero value for structs is a struct with all fields set to their zero values
	var statsJSON types.StatsJSON

	// json.NewDecoder creates a decoder that reads from stats.Body
	// Method chaining: NewDecoder().Decode() calls Decode on the decoder
	if err := json.NewDecoder(stats.Body).Decode(&statsJSON); err != nil {
		return nil, err
	}

	// &statsJSON takes the address of the statsJSON variable
	// This returns a pointer to the struct
	return &statsJSON, nil
}

// calculateCPUPercent calculates CPU usage percentage from Docker stats.
// This method demonstrates Go's approach to numeric calculations.
func (m *Monitor) calculateCPUPercent(stats *types.StatsJSON) float64 {
	// Type conversion from uint64 to float64 for precise calculations
	// Go requires explicit type conversions - no automatic conversions
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage) - float64(stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage) - float64(stats.PreCPUStats.SystemUsage)

	// Conditional logic with boolean operators
	// && is logical AND, > is greater than comparison
	if systemDelta > 0 && cpuDelta > 0 {
		// len() returns the length of a slice (dynamic array)
		// Type conversion of int to float64 for calculation
		// Mathematical formula for CPU percentage calculation
		return (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}

	// Return zero if calculation isn't possible
	return 0.0
}

// formatBytes converts byte counts to human-readable format (KB, MB, GB, etc.)
// uint64 is an unsigned 64-bit integer type
func (m *Monitor) formatBytes(bytes uint64) string {
	// const declares a compile-time constant
	// Unlike var, const values must be known at compile time
	const unit = 1024

	// Early return pattern - handle simple case first
	if bytes < unit {
		// fmt.Sprintf is like printf - returns formatted string without printing
		// %d is format verb for decimal integers
		return fmt.Sprintf("%d B", bytes)
	}

	// Multiple variable declaration in one line
	// int64 is a signed 64-bit integer
	div, exp := int64(unit), 0

	// for loop with multiple conditions
	// := in loop declares and assigns n variable
	// /= is compound assignment operator (equivalent to: n = n / unit)
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit // Multiply-assign operator
		exp++       // Increment operator
	}

	// %.1f formats float with 1 decimal place
	// %c formats a character (using ASCII/Unicode value)
	// String indexing: "KMGTPE"[exp] gets character at position exp
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// isFromSameComposeProject checks if a container belongs to our Docker Compose project.
// map[string]string is a hash map (dictionary) with string keys and string values.
func (m *Monitor) isFromSameComposeProject(labels map[string]string) bool {
	// Slice literal - []string{...} creates a slice of strings
	// Slices are dynamic arrays that can grow and shrink
	composeLabels := []string{
		"com.docker.compose.project",
		"com.docker.compose.project.name",
		"docker-compose.project",
	}

	// range keyword iterates over slices, arrays, maps, etc.
	// _ ignores the index, label gets the value
	for _, label := range composeLabels {
		// Map lookup with "comma ok" idiom
		// project gets the value, exists is a boolean indicating if key was found
		if project, exists := labels[label]; exists {
			// String comparison
			return project == m.project
		}
	}

	// Alternative check using different label
	if containerName, exists := labels["com.docker.compose.service"]; exists {
		// strings.HasPrefix checks if string starts with given prefix
		return strings.HasPrefix(containerName, m.project)
	}

	// Default return - function must return a value on all paths
	return false
}

// monitorContainers is the core monitoring function that gathers container metrics.
func (m *Monitor) monitorContainers(ctx context.Context) error {
	// Call Docker API to list all containers (running and stopped)
	// container.ListOptions is a struct that configures the API call
	containers, err := m.client.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		// Error wrapping with context information
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Declare a slice of ContainerMetrics structs
	// nil is the zero value for slices, maps, pointers, interfaces
	var metrics []logger.ContainerMetrics

	// Range over slice of containers
	// container is each individual container object
	for _, container := range containers {
		// Method call on receiver - check if container belongs to our project
		if !m.isFromSameComposeProject(container.Labels) {
			// continue skips to next iteration of the loop
			continue
		}

		// Struct literal initialization with field names
		// This creates a new ContainerMetrics struct
		metric := logger.ContainerMetrics{
			// strings.TrimPrefix removes leading "/" from container name
			Name:      strings.TrimPrefix(container.Names[0], "/"),
			Status:    container.Status,    // Direct field assignment
			Image:     container.Image,     // Container image name
			Timestamp: time.Now(),          // Current timestamp
			Labels:    container.Labels,    // Copy the entire labels map
			Project:   m.project,           // Our project name
		}

		// Conditional execution based on container state
		if container.State == "running" {
			// Method call that might return an error
			stats, err := m.getContainerStats(ctx, container.ID)
			if err != nil {
				// Log error but continue processing other containers
				// fmt.Sprintf creates formatted string for error message
				m.logManager.LogError(fmt.Sprintf("Failed to get stats for container %s", metric.Name), err)
			} else {
				// Update metric fields with calculated values
				metric.CPUPercent = m.calculateCPUPercent(stats)
				metric.MemoryUsage = m.formatBytes(stats.MemoryStats.Usage)
				metric.MemoryLimit = m.formatBytes(stats.MemoryStats.Limit)
			}
		}

		// append() adds elements to a slice and returns the new slice
		// Go slices automatically grow when needed
		metrics = append(metrics, metric)
	}

	// Delegate logging to the log manager
	// This will write to all configured logging drivers
	return m.logManager.LogMetrics(metrics)
}

// Start begins the monitoring loop and runs until context is cancelled.
func (m *Monitor) Start(ctx context.Context) error {
	// Log startup message with structured data
	// map[string]interface{} allows mixed types as values
	// interface{} is Go's "any type" - similar to Object in Java
	m.logManager.LogInfo("Starting Docker container monitor", map[string]interface{}{
		"project":   m.project,            // string value
		"frequency": m.frequency.String(), // duration as string
	})

	// time.NewTicker creates a ticker that sends on its channel at regular intervals
	// Channels are Go's way of communicating between goroutines
	ticker := time.NewTicker(m.frequency)

	// defer ensures ticker.Stop() is called when function exits
	// This prevents resource leaks
	defer ticker.Stop()

	// Perform initial monitoring before entering the loop
	if err := m.monitorContainers(ctx); err != nil {
		m.logManager.LogError("Initial monitoring failed", err)
		return err // Return early on error
	}

	// Infinite loop for continuous monitoring
	for {
		// select statement chooses between multiple channel operations
		// It's similar to switch but for channels
		select {
		// <-ctx.Done() receives from the context's done channel
		// This channel is closed when context is cancelled
		case <-ctx.Done():
			m.logManager.LogInfo("Monitor stopping due to context cancellation", nil)
			m.logManager.Close() // Clean up resources
			return ctx.Err()     // Return the context's error

		// <-ticker.C receives from the ticker's channel
		// This triggers every m.frequency duration
		case <-ticker.C:
			// Run monitoring cycle, but don't stop on errors
			if err := m.monitorContainers(ctx); err != nil {
				m.logManager.LogError("Monitoring cycle failed", err)
				// Continue the loop instead of returning
			}
		}
	}
}

// main is the entry point of the program.
// Every Go executable must have exactly one main() function in a main package.
func main() {
	// Call constructor function to create Monitor instance
	monitor, err := NewMonitor()
	if err != nil {
		// fmt.Printf prints formatted output to standard output
		// %v is the default format verb that works with any type
		fmt.Printf("Failed to create monitor: %v\n", err)

		// os.Exit terminates the program with given exit code
		// Exit code 1 indicates an error (0 would mean success)
		os.Exit(1)
	}

	// context.Background() creates an empty, never-cancelled context
	// This is commonly used as the root context for applications
	ctx := context.Background()

	// Start the monitoring process
	if err := monitor.Start(ctx); err != nil {
		// Log the error before exiting
		monitor.logManager.LogError("Monitor failed", err)
		monitor.logManager.Close() // Clean up logging resources
		os.Exit(1)                  // Exit with error code
	}
}

/*
Key Go Language Concepts Demonstrated:

1. **Package System**: Code organization with main package and imports
2. **Struct Types**: Custom data types that group related data
3. **Methods**: Functions that operate on specific types (receivers)
4. **Pointers**: References to memory addresses (*, &)
5. **Error Handling**: Explicit error checking with multiple return values
6. **Slices**: Dynamic arrays that can grow and shrink
7. **Maps**: Hash tables/dictionaries for key-value storage
8. **Channels**: Communication mechanism for concurrent programming
9. **Interfaces**: Contracts that types can implement (interface{})
10. **Goroutines**: Lightweight threads (not used here but ticker uses them)
11. **Defer**: Resource cleanup that runs when function exits
12. **Select**: Choose between multiple channel operations
13. **Type Conversion**: Explicit conversion between types
14. **String Operations**: Manipulation and formatting of text
15. **Time Handling**: Duration and timestamp operations
16. **JSON Operations**: Encoding and decoding structured data
17. **File Operations**: Reading files and environment variables
18. **Context**: Managing request lifecycles and cancellation
19. **Constants**: Compile-time constant values
20. **Zero Values**: Default values for uninitialized variables
*/