// Package logger provides multiple logging drivers for the Docker monitor application.
// This file implements the LogManager which orchestrates multiple logging drivers simultaneously.
// It demonstrates Go's composition and polymorphism patterns.
package logger

// Import statements bring in required packages for the LogManager functionality.
// Go groups imports by standard library, then external packages, then local packages.
import (
	// Standard library packages (built into Go)
	"fmt"           // Formatted I/O functions for error messages and console output
	"os"            // Operating system interface for file operations and environment variables
	"path/filepath" // File path manipulation utilities
	"strings"       // String manipulation functions like ToLower()

	// External packages for YAML support
	"gopkg.in/yaml.v3" // YAML v3 encoding/decoding library
)

// LogManager is a struct that implements the manager pattern (also called orchestrator pattern).
// It manages multiple LogDriver implementations and routes operations to all of them.
// This demonstrates composition over inheritance - Go doesn't have inheritance, so we compose behavior.
type LogManager struct {
	// Struct fields define the data that each LogManager instance will contain

	// drivers is a slice (dynamic array) of LogDriver interfaces
	// []LogDriver means "slice of LogDriver interface implementations"
	// Interface slices allow polymorphism - we can store different concrete types
	// that all implement the LogDriver interface
	drivers []LogDriver

	// configs is a slice of LogConfig structs that correspond to each driver
	// This maintains the original configuration for each driver for reference
	// The slice indices align: drivers[0] corresponds to configs[0], etc.
	configs []LogConfig
}

// NewLogManager is a constructor function that creates and initializes a new LogManager.
// In Go, constructor functions are the idiomatic way to create objects since there are no classes.
// The function name starts with "New" by convention and returns a pointer to the struct.
func NewLogManager() *LogManager {
	// Return a pointer to a newly created LogManager struct
	// &LogManager{...} creates a struct literal and returns its memory address
	return &LogManager{
		// make() creates a slice with specified type, length, and capacity
		// make([]LogDriver, 0) creates an empty slice of LogDriver interfaces
		// 0 is the initial length (no elements), capacity defaults to length
		drivers: make([]LogDriver, 0),

		// Similarly create an empty slice for configs
		// These slices will grow dynamically as we add drivers
		configs: make([]LogConfig, 0),
	}
}

// LoadConfigFromEnv loads logging configuration from environment variables.
// This is a method on LogManager (lm is the receiver variable).
// Methods in Go are functions with a special receiver argument before the function name.
func (lm *LogManager) LoadConfigFromEnv() error {
	// os.Getenv reads environment variables from the operating system
	// It returns a string value, or empty string if the variable doesn't exist
	configStr := os.Getenv("LOG_CONFIG")

	// String comparison in Go - check if the environment variable is empty
	// Go's zero value for strings is "" (empty string)
	if configStr == "" {
		// Default behavior when no configuration is provided
		// Create a default console driver with JSON formatting

		// Method call on the receiver (lm) - calling our own AddDriver method
		// LogConfig{...} creates a struct literal with field initialization
		lm.AddDriver(LogConfig{
			Driver: DriverConsole, // DriverConsole is a constant defined in types.go

			// map[string]string{...} creates a map literal with string keys and values
			// Maps in Go are hash tables (dictionaries) that store key-value pairs
			Options: map[string]string{
				"format": "json", // Key-value pair for output format
				"level":  "info", // Key-value pair for log level
			},
		})

		// Return nil to indicate successful completion
		// nil is Go's zero value for pointers, interfaces, maps, slices, channels, functions
		return nil
	}

	// Parse the configuration string as YAML
	// var declares a variable with its zero value
	// []LogConfig is a slice of LogConfig structs
	var configs []LogConfig

	// Parse the YAML configuration
	if err := yaml.Unmarshal([]byte(configStr), &configs); err != nil {
		return fmt.Errorf("failed to parse LOG_CONFIG as YAML: %w", err)
	}

	// Loop through each configuration to add corresponding drivers
	// range keyword iterates over slices, arrays, maps, channels
	// _ (blank identifier) ignores the index, config gets each LogConfig value
	for _, config := range configs {
		// Method call that might return an error
		// We check the error immediately (Go's explicit error handling)
		if err := lm.AddDriver(config); err != nil {
			// Error wrapping with additional context
			// %s format verb for strings, config.Driver is the driver name
			return fmt.Errorf("failed to add driver %s: %w", config.Driver, err)
		}
	}

	// Return nil indicating successful processing of all configurations
	return nil
}

// LoadConfigFromFile loads logging configuration from a YAML file.
// This method demonstrates file I/O operations in Go.
// filePath parameter is the path to the configuration file.
func (lm *LogManager) LoadConfigFromFile(filePath string) error {
	// os.ReadFile reads the entire file contents into memory
	// It returns []byte (byte slice) and error (multiple return values pattern)
	// This is Go's modern way of reading files (replaces ioutil.ReadFile)
	data, err := os.ReadFile(filePath)

	// Explicit error checking - always check errors in Go
	if err != nil {
		// Error wrapping with context about which operation failed
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Declare a variable to hold the parsed configuration
	// Zero value for slices is nil (no allocated memory)
	var configs []LogConfig

	// Parse the configuration file as YAML
	if err := yaml.Unmarshal(data, &configs); err != nil {
		return fmt.Errorf("failed to parse YAML config file: %w", err)
	}

	// Process each configuration - same logic as LoadConfigFromEnv
	// This demonstrates code reuse through method calls
	for _, config := range configs {
		if err := lm.AddDriver(config); err != nil {
			return fmt.Errorf("failed to add driver %s: %w", config.Driver, err)
		}
	}

	return nil
}

// AddDriver creates and initializes a new logging driver based on configuration.
// This method demonstrates the factory pattern - creating objects based on type.
// config parameter contains the driver type and configuration options.
func (lm *LogManager) AddDriver(config LogConfig) error {
	// var declares a variable with its zero value
	// LogDriver is an interface, so zero value is nil
	// This variable will hold a concrete implementation of the LogDriver interface
	var driver LogDriver

	// Defensive programming - ensure Options map is initialized
	// nil check for maps - maps can be nil in Go
	if config.Options == nil {
		// make() creates an empty map with string keys and string values
		// This prevents nil pointer panics when accessing the map
		config.Options = make(map[string]string)
	}

	// switch statement for pattern matching on driver type
	// strings.ToLower() converts string to lowercase for case-insensitive matching
	// This demonstrates string manipulation and conditional logic
	switch strings.ToLower(config.Driver) {
	// case statements check for specific values
	// DriverConsole, DriverCSV, etc. are constants defined in types.go
	case DriverConsole:
		// Constructor function call - creates ConsoleDriver instance
		// NewConsoleDriver returns a *ConsoleDriver which implements LogDriver interface
		// Interface assignment - concrete type assigned to interface variable
		driver = NewConsoleDriver(config)

	case DriverCSV:
		// Same pattern for CSV driver
		driver = NewCSVDriver(config)

	case DriverExcel:
		// Same pattern for Excel driver
		driver = NewExcelDriver(config)

	case DriverCloudWatch:
		// Same pattern for CloudWatch driver
		driver = NewCloudWatchDriver(config)

	default:
		// default case handles unsupported driver types
		// fmt.Errorf creates a custom error with formatted message
		// %s format verb inserts the driver name string
		return fmt.Errorf("unsupported driver: %s", config.Driver)
	}

	// Initialize the driver (setup resources, connections, etc.)
	// Method call on interface - polymorphic call that works for any LogDriver implementation
	// Each driver type implements Initialize() differently
	if err := driver.Initialize(); err != nil {
		// Error wrapping with context about which driver failed
		return fmt.Errorf("failed to initialize driver %s: %w", config.Driver, err)
	}

	// Add the successfully initialized driver to our collections
	// append() adds elements to a slice and returns the new slice
	// Go slices are dynamic arrays that automatically grow when needed
	lm.drivers = append(lm.drivers, driver)

	// Store the configuration for reference (same index as driver)
	// This maintains the correspondence between drivers and their configs
	lm.configs = append(lm.configs, config)

	// Return nil indicating successful driver addition
	return nil
}

// LogMetrics routes container metrics to all configured logging drivers.
// This method demonstrates the fan-out pattern - one input, multiple outputs.
// metrics parameter is a slice of ContainerMetrics to be logged.
func (lm *LogManager) LogMetrics(metrics []ContainerMetrics) error {
	// Variable to track the last error encountered
	// Go pattern for collecting errors from multiple operations
	// error is an interface type in Go
	var lastErr error

	// Iterate through drivers with index and value
	// range with two variables gives index (i) and value (driver)
	// We need the index to access the corresponding config
	for i, driver := range lm.drivers {
		// Polymorphic method call - each driver implements LogMetrics differently
		// Interface method call works regardless of the concrete type
		if err := driver.LogMetrics(metrics); err != nil {
			// Store the error for potential return
			lastErr = err

			// Error handling with conditional logic
			// Access configs slice using the same index as drivers slice
			// != is the "not equal" comparison operator
			if lm.configs[i].Driver != DriverConsole {
				// fmt.Printf prints formatted output to standard output
				// %s format verb for strings, %v format verb for any value
				// This provides fallback error reporting when other drivers fail
				fmt.Printf("Error logging to %s driver: %v\n", lm.configs[i].Driver, err)
			}
		}
	}

	// Return the last error encountered (nil if no errors)
	// This allows the caller to know if any driver failed
	return lastErr
}

// LogInfo routes informational messages to all configured logging drivers.
// This method demonstrates the same fan-out pattern for info messages.
// message is the main log message, fields contains additional structured data.
func (lm *LogManager) LogInfo(message string, fields map[string]interface{}) error {
	// Same error collection pattern as LogMetrics
	var lastErr error

	// Iterate through all drivers
	// range with one variable gives only the value (driver), ignoring index
	// We use this when we don't need the index
	for _, driver := range lm.drivers {
		// Polymorphic method call to LogInfo
		// Each driver handles info logging according to its implementation
		if err := driver.LogInfo(message, fields); err != nil {
			// Collect errors but continue processing other drivers
			lastErr = err
		}
	}

	return lastErr
}

// LogError routes error messages to all configured logging drivers.
// This method shows how the same pattern applies to error logging.
// message describes what failed, err contains the actual error details.
func (lm *LogManager) LogError(message string, err error) error {
	var lastErr error

	for _, driver := range lm.drivers {
		// Variable shadowing - driverErr shadows the err parameter
		// This is legal in Go due to lexical scoping
		// We use different variable name to avoid confusion
		if driverErr := driver.LogError(message, err); driverErr != nil {
			lastErr = driverErr
		}
	}

	return lastErr
}

// Close gracefully shuts down all logging drivers and cleans up resources.
// This method demonstrates the cleanup pattern in Go.
func (lm *LogManager) Close() error {
	var lastErr error

	// Close all drivers to release resources (files, connections, etc.)
	for _, driver := range lm.drivers {
		// Polymorphic call to Close method
		// Each driver implements cleanup according to its needs
		if err := driver.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// GetDriverCount returns the number of active logging drivers.
// This is a simple getter method that demonstrates slice length operations.
func (lm *LogManager) GetDriverCount() int {
	// len() is a built-in function that returns the length of slices, arrays, maps, strings, channels
	// It returns an int value representing the number of elements
	return len(lm.drivers)
}

// GetDriverTypes returns a slice of driver type names.
// This method demonstrates slice creation and population patterns.
func (lm *LogManager) GetDriverTypes() []string {
	// Pre-allocate slice with known size for efficiency
	// make([]string, len(lm.configs)) creates a slice with length equal to configs count
	// This avoids multiple memory allocations during append operations
	types := make([]string, len(lm.configs))

	// Iterate with index and value to populate the new slice
	// range provides both index (i) and value (config)
	for i, config := range lm.configs {
		// Direct slice assignment using index
		// types[i] accesses the slice element at index i
		// config.Driver accesses the Driver field of the LogConfig struct
		types[i] = config.Driver
	}

	// Return the populated slice
	return types
}

// CreateSampleConfig creates a sample configuration file with all supported drivers.
// This is a package-level function (no receiver) that generates YAML format.
func CreateSampleConfig(filePath string) error {
	// Create a slice literal with sample configurations for all driver types
	// []LogConfig{...} creates and initializes a slice of LogConfig structs
	sampleConfigs := []LogConfig{
		// Each {...} is a struct literal creating a LogConfig instance
		{
			// Named field initialization - clearer than positional initialization
			Driver:     DriverConsole, // Console driver configuration
			OutputPath: "",            // Empty path for console (outputs to stdout)

			// Map literal with string keys and values
			Options: map[string]string{
				"format": "json", // JSON output format
				"level":  "info", // Info log level
			},
		},
		{
			Driver:     DriverCSV,                         // CSV driver configuration
			OutputPath: "/logs/container_metrics.csv",    // File path for CSV output
			Options: map[string]string{
				"info_log_file":  "/logs/info.log",    // Separate file for info logs
				"error_log_file": "/logs/errors.log",  // Separate file for error logs
			},
		},
		{
			Driver:     DriverExcel,                       // Excel driver configuration
			OutputPath: "/logs/container_metrics.xlsx",   // File path for Excel output
			Options: map[string]string{
				"sheet_name": "Container Metrics", // Excel worksheet name
			},
		},
		{
			Driver:     DriverCloudWatch,                  // CloudWatch driver configuration
			OutputPath: "",                               // No local path for cloud service
			Options: map[string]string{
				"log_group":      "/docker-monitor/container-metrics", // CloudWatch log group
				"log_stream":     "container-monitor",                 // CloudWatch log stream
				"region":         "us-east-1",                        // AWS region
				"retention_days": "30",                               // Log retention period
			},
		},
	}

	// Generate YAML format - human-readable and cleaner
	// yaml.Marshal converts Go data structures to YAML bytes
	data, err := yaml.Marshal(sampleConfigs)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// os.WriteFile writes data to a file, creating it if necessary
	// filePath is the target file path
	// data contains YAML bytes
	// 0644 is the file permissions (owner read/write, group/others read-only)
	return os.WriteFile(filePath, data, 0644)
}

/*
Key Go Language Concepts Demonstrated in LogManager:

1. **Struct Composition**: LogManager composes behavior from multiple LogDriver interfaces
2. **Interface Polymorphism**: []LogDriver slice holds different concrete implementations
3. **Method Receivers**: Functions that operate on struct instances (lm *LogManager)
4. **Constructor Pattern**: NewLogManager() function creates and initializes structs
5. **Error Handling**: Explicit error checking and wrapping throughout
6. **Slice Operations**: Dynamic arrays with make(), append(), len(), range
7. **Map Operations**: Hash tables for configuration options
8. **YAML Marshaling/Unmarshaling**: Converting between Go structs and YAML
9. **File I/O**: Reading and writing files with os.ReadFile/WriteFile
10. **Environment Variables**: Reading configuration from os.Getenv()
11. **String Operations**: Case conversion and formatting
12. **Switch Statements**: Pattern matching for driver type selection
13. **Factory Pattern**: Creating objects based on configuration type
14. **Fan-out Pattern**: Distributing operations to multiple receivers
15. **Resource Management**: Cleanup with Close() methods
16. **Zero Values**: Understanding nil for interfaces, slices, maps
17. **Variable Shadowing**: Reusing variable names in different scopes
18. **Multiple Return Values**: Functions returning (value, error) pairs
19. **Slice Preallocation**: Performance optimization with make([]T, size)
20. **Package-level Functions**: Functions without receivers for utilities

Design Patterns Demonstrated:

1. **Manager/Orchestrator Pattern**: LogManager coordinates multiple drivers
2. **Factory Pattern**: AddDriver creates drivers based on configuration
3. **Strategy Pattern**: Different drivers implement different logging strategies
4. **Composite Pattern**: LogManager treats single drivers and collections uniformly
5. **Configuration Pattern**: External configuration drives behavior
6. **Error Aggregation Pattern**: Collecting errors from multiple operations
7. **Resource Management Pattern**: Initialize/Close lifecycle management
8. **Fallback Pattern**: Console output when other drivers fail
9. **Polymorphic Dispatch**: Interface methods called on concrete implementations
10. **Lazy Initialization**: Drivers created only when needed

Performance Considerations:

1. **Slice Preallocation**: make([]T, size) avoids reallocations
2. **Interface Method Calls**: Small overhead but enables polymorphism
3. **Error Collection**: Continue processing despite individual failures
4. **Memory Management**: Go's garbage collector handles cleanup
5. **YAML Operations**: Efficient encoding/decoding of configuration
*/