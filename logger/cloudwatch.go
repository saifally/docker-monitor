// Package logger provides multiple logging drivers for the Docker monitor application.
// This file specifically implements the CloudWatch logging driver that sends logs to AWS CloudWatch Logs.
package logger

// Import statements bring in required packages.
// Go organizes functionality into packages - collections of related functions and types.
import (
	// Standard library packages (built into Go)
	"context"       // Provides context for managing timeouts, cancellation, and request-scoped values
	"encoding/json" // JSON encoding/decoding functionality for structured data
	"fmt"           // Formatted I/O functions (like printf in C)
	"strconv"       // String conversion utilities (string to int, etc.)
	"time"          // Time and duration handling

	// AWS SDK v2 packages (external dependencies)
	// These packages provide AWS service clients and utilities
	"github.com/aws/aws-sdk-go-v2/aws"                            // Core AWS SDK types and utilities
	"github.com/aws/aws-sdk-go-v2/config"                         // AWS configuration loading (credentials, region, etc.)
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"         // CloudWatch Logs service client
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"   // CloudWatch Logs specific types and structures
)

// CloudWatchDriver is a struct that implements the LogDriver interface.
// It encapsulates all data and behavior needed to send logs to AWS CloudWatch Logs.
// Structs in Go are similar to classes in other languages but without inheritance.
type CloudWatchDriver struct {
	// Struct fields define the data that each CloudWatchDriver instance will contain

	// config stores the logging configuration passed during driver creation
	// LogConfig is a custom type defined in types.go
	config LogConfig

	// client is a pointer to the AWS CloudWatch Logs service client
	// *cloudwatchlogs.Client means "pointer to a CloudWatch Logs client"
	// Pointers store memory addresses rather than copying entire structures
	client *cloudwatchlogs.Client

	// logGroupName is the AWS CloudWatch Log Group name where logs will be sent
	// string is Go's built-in string type (UTF-8 encoded)
	logGroupName string

	// logStreamName is the specific log stream within the log group
	// Log streams organize logs chronologically within a log group
	logStreamName string

	// sequenceToken is required by AWS CloudWatch for ordering log events
	// *string means "pointer to string" - can be nil if no token exists yet
	// AWS uses sequence tokens to ensure log events are processed in order
	sequenceToken *string
}

// NewCloudWatchDriver is a constructor function that creates and initializes a new CloudWatchDriver.
// In Go, constructor functions are a common pattern since there are no class constructors.
// The function takes a LogConfig parameter and returns a pointer to CloudWatchDriver.
func NewCloudWatchDriver(config LogConfig) *CloudWatchDriver {
	// Set default log group name
	// := is Go's short variable declaration - declares and assigns in one step
	// The type (string) is inferred from the right-hand side
	logGroupName := "/docker-monitor/container-metrics"

	// Map lookup with "comma ok" idiom - a Go pattern for safe map access
	// group gets the value from the map, exists is a boolean indicating if the key was found
	// config.Options is a map[string]string defined in LogConfig
	if group, exists := config.Options["log_group"]; exists {
		// Only update logGroupName if the key exists in the map
		logGroupName = group
	}

	// Generate a unique log stream name using current Unix timestamp
	// fmt.Sprintf is like printf - returns a formatted string without printing
	// %d is a format verb for decimal integers
	// time.Now().Unix() returns current time as Unix timestamp (seconds since epoch)
	logStreamName := fmt.Sprintf("container-monitor-%d", time.Now().Unix())

	// Check if a custom log stream name was provided in configuration
	if stream, exists := config.Options["log_stream"]; exists {
		logStreamName = stream
	}

	// Return a pointer to a newly created CloudWatchDriver struct
	// &CloudWatchDriver{...} creates a struct literal and returns its memory address
	// This is Go's way of creating objects on the heap
	return &CloudWatchDriver{
		config:        config,        // Store the entire configuration
		logGroupName:  logGroupName,  // Store the determined log group name
		logStreamName: logStreamName, // Store the determined log stream name
		// sequenceToken is not set here - it starts as nil (zero value for pointers)
		// client is also not set here - it will be initialized in Initialize()
	}
}

// Initialize sets up the CloudWatch client and creates necessary AWS resources.
// This is a method on CloudWatchDriver (c is the receiver variable).
// Methods in Go are functions with a special receiver argument.
func (c *CloudWatchDriver) Initialize() error {
	// Load AWS configuration from environment variables, shared credentials file, etc.
	// context.TODO() creates a non-nil, empty Context for when you don't have a specific context
	// config.LoadDefaultConfig automatically discovers AWS credentials and configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())

	// Go's explicit error handling pattern - always check if err is not nil
	if err != nil {
		// fmt.Errorf creates a formatted error with error wrapping
		// %w verb wraps the original error (allows error unwrapping)
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Override the AWS region if specified in the driver configuration
	// This allows users to specify a different region than their default AWS config
	if region, exists := c.config.Options["region"]; exists {
		// Directly assign to the config struct field
		// cfg.Region is a string field in the AWS config
		cfg.Region = region
	}

	// Create a new CloudWatch Logs client using the loaded configuration
	// cloudwatchlogs.NewFromConfig creates a service client from AWS config
	// The client handles authentication, retries, and AWS API communication
	c.client = cloudwatchlogs.NewFromConfig(cfg)

	// Create the log group if it doesn't already exist
	// This is a method call on the receiver (c) - calling our own method
	if err := c.createLogGroupIfNotExists(); err != nil {
		// Error wrapping - add context to the error before returning it
		return fmt.Errorf("failed to create log group: %w", err)
	}

	// Create the log stream within the log group
	if err := c.createLogStream(); err != nil {
		return fmt.Errorf("failed to create log stream: %w", err)
	}

	// Return nil to indicate successful initialization
	// In Go, nil is the zero value for pointers, interfaces, maps, slices, channels, and function types
	return nil
}

// LogMetrics sends container metrics to CloudWatch Logs as structured JSON events.
// This method implements the LogDriver interface requirement.
// It takes a slice of ContainerMetrics and returns an error if sending fails.
func (c *CloudWatchDriver) LogMetrics(metrics []ContainerMetrics) error {
	// make() creates a slice with specified length and capacity
	// []types.InputLogEvent is the slice type (slice of InputLogEvent structs)
	// 0 is the initial length (empty slice)
	// len(metrics)+1 is the capacity (pre-allocate space for efficiency)
	// This avoids memory reallocations as we append events
	events := make([]types.InputLogEvent, 0, len(metrics)+1)

	// Create a summary event that aggregates information about this monitoring cycle
	// map[string]interface{} is a map with string keys and values of any type
	// interface{} is Go's "any type" - similar to Object in Java or any in TypeScript
	summaryData := map[string]interface{}{
		"event_type":       "monitoring_summary", // String literal
		"containers_count": len(metrics),          // len() returns int - number of containers
		"timestamp":        time.Now(),           // Current timestamp
	}

	// json.Marshal converts Go data structures to JSON bytes
	// It returns []byte and error (multiple return values - common Go pattern)
	// _ (blank identifier) discards the error - we're assuming marshal won't fail for simple data
	summaryJSON, _ := json.Marshal(summaryData)

	// append() adds elements to a slice and returns the new slice
	// Go slices are dynamic arrays that automatically grow when needed
	// types.InputLogEvent is an AWS SDK struct for CloudWatch log events
	events = append(events, types.InputLogEvent{
		// aws.String() converts string to *string (pointer to string)
		// AWS SDK uses pointers to distinguish between nil and empty strings
		Message: aws.String(string(summaryJSON)), // Convert []byte to string, then to *string

		// aws.Int64() converts int64 to *int64 (pointer to int64)
		// time.Now().UnixMilli() returns current time as milliseconds since Unix epoch
		// CloudWatch expects timestamps in milliseconds
		Timestamp: aws.Int64(time.Now().UnixMilli()),
	})

	// Loop through each container metric to create individual log events
	// range keyword iterates over slices, arrays, maps, etc.
	// _ ignores the index, metric gets each ContainerMetrics value
	for _, metric := range metrics {
		// Create structured log data for each container
		// This map contains all the container information we want to log
		logData := map[string]interface{}{
			"event_type":     "container_metrics", // Classify this as a metrics event
			"container_name": metric.Name,          // Container name from Docker
			"status":         metric.Status,        // Container status (running, stopped, etc.)
			"image":          metric.Image,         // Docker image name
			"cpu_percent":    metric.CPUPercent,    // CPU usage percentage
			"memory_usage":   metric.MemoryUsage,   // Memory usage in human-readable format
			"memory_limit":   metric.MemoryLimit,   // Memory limit in human-readable format
			"project":        metric.Project,       // Docker Compose project name
			"timestamp":      metric.Timestamp,     // When this metric was collected
		}

		// Add Docker Compose service name if it exists in the container labels
		// This demonstrates conditional map population based on data availability
		if service, exists := metric.Labels["com.docker.compose.service"]; exists {
			// Add the service name to our log data
			logData["compose_service"] = service
		}

		// Add other useful Docker Compose labels for debugging and analysis
		// range over map returns key, value pairs
		for key, value := range metric.Labels {
			// Multi-condition if statement using logical OR (||)
			// Check if the label key matches any of our interesting labels
			if key == "com.docker.compose.version" ||
				key == "com.docker.compose.config-hash" ||
				key == "com.docker.compose.container-number" {
				// Add this label to our log data
				logData[key] = value
			}
		}

		// Convert the log data map to JSON
		// This time we check the error since malformed data could cause issues
		logJSON, err := json.Marshal(logData)
		if err != nil {
			// continue skips the rest of this loop iteration and moves to the next one
			// This ensures one bad metric doesn't stop us from logging others
			continue // Skip malformed entries
		}

		// Add this container's metrics as a log event
		events = append(events, types.InputLogEvent{
			Message:   aws.String(string(logJSON)),                    // Convert JSON bytes to string, then pointer
			Timestamp: aws.Int64(metric.Timestamp.UnixMilli()),       // Use the metric's timestamp
		})
	}

	// Send all events to CloudWatch in a batch
	// This calls our private helper method to handle the AWS API interaction
	return c.putLogEvents(events)
}

// LogInfo sends informational log messages to CloudWatch.
// This method handles general application info like startup, shutdown, etc.
// message is the main log message, fields contains additional structured data.
func (c *CloudWatchDriver) LogInfo(message string, fields map[string]interface{}) error {
	// Create base log data structure for info events
	logData := map[string]interface{}{
		"event_type": "info",       // Classify this as an info event
		"message":    message,      // The main log message
		"timestamp":  time.Now(),   // When this event occurred
	}

	// Add any additional fields provided by the caller
	// This loop copies all key-value pairs from fields map to logData map
	// range over map provides key (k) and value (v) on each iteration
	for k, v := range fields {
		// Direct assignment adds/overwrites the key in the destination map
		logData[k] = v
	}

	// Convert to JSON for CloudWatch
	logJSON, err := json.Marshal(logData)
	if err != nil {
		// Return the error immediately if JSON marshaling fails
		return err
	}

	// Create a slice literal with one event
	// []types.InputLogEvent{...} creates a slice and initializes it with one element
	events := []types.InputLogEvent{
		{
			Message:   aws.String(string(logJSON)),
			Timestamp: aws.Int64(time.Now().UnixMilli()),
		},
	}

	// Send the single event to CloudWatch
	return c.putLogEvents(events)
}

// LogError sends error log messages to CloudWatch.
// This method handles application errors, API failures, etc.
// message describes what operation failed, err contains the actual error details.
func (c *CloudWatchDriver) LogError(message string, err error) error {
	// Create structured error log data
	logData := map[string]interface{}{
		"event_type": "error",        // Classify this as an error event
		"message":    message,        // Description of what failed
		"error":      err.Error(),    // Convert error to string using Error() method
		"timestamp":  time.Now(),     // When this error occurred
	}

	// Convert to JSON
	// Variable name jsonErr avoids conflict with the err parameter
	// This demonstrates Go's lexical scoping - variables can shadow outer scope variables
	logJSON, jsonErr := json.Marshal(logData)
	if jsonErr != nil {
		// Return the JSON marshaling error, not the original error
		return jsonErr
	}

	// Create event slice
	events := []types.InputLogEvent{
		{
			Message:   aws.String(string(logJSON)),
			Timestamp: aws.Int64(time.Now().UnixMilli()),
		},
	}

	// Send to CloudWatch
	return c.putLogEvents(events)
}

// Close sends a final shutdown event to CloudWatch and cleans up resources.
// This method is called when the application is shutting down.
func (c *CloudWatchDriver) Close() error {
	// Create a shutdown event to mark the end of this monitoring session
	logData := map[string]interface{}{
		"event_type": "monitor_shutdown",              // Classify this as a shutdown event
		"message":    "Docker monitor shutting down", // Human-readable shutdown message
		"timestamp":  time.Now(),                      // When shutdown occurred
	}

	// Convert to JSON
	// Again using _ to ignore the error since this is simple data
	logJSON, _ := json.Marshal(logData)

	// Create the shutdown event
	events := []types.InputLogEvent{
		{
			Message:   aws.String(string(logJSON)),
			Timestamp: aws.Int64(time.Now().UnixMilli()),
		},
	}

	// Send the shutdown event
	// We call putLogEvents but ignore its return value since we're shutting down
	c.putLogEvents(events)

	// Always return nil from Close() - we don't want shutdown to fail
	return nil
}

// createLogGroupIfNotExists creates a CloudWatch Log Group if it doesn't already exist.
// This is a private method (lowercase name) that handles AWS resource creation.
func (c *CloudWatchDriver) createLogGroupIfNotExists() error {
	// Create a context for the AWS API call
	// context.TODO() is used when we don't have a specific context to pass
	ctx := context.TODO()

	// Check if the log group already exists by describing it
	// DescribeLogGroups lists log groups matching the specified prefix
	// We use the prefix to check if our specific log group exists
	_, err := c.client.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{
		// &cloudwatchlogs.DescribeLogGroupsInput{...} creates a struct literal and takes its address
		// This is required because the AWS SDK expects a pointer to the input struct
		LogGroupNamePrefix: aws.String(c.logGroupName),
	})

	// If no error, the log group exists
	if err == nil {
		return nil // Log group exists, nothing to do
	}

	// Create the log group since it doesn't exist
	// CreateLogGroup is an AWS CloudWatch Logs API operation
	_, err = c.client.CreateLogGroup(ctx, &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(c.logGroupName),
	})

	// Handle potential errors from log group creation
	if err != nil {
		// Type assertion with "comma ok" idiom
		// This checks if err is specifically a "ResourceAlreadyExistsException"
		// _, ok := err.(*types.ResourceAlreadyExistsException) performs type assertion
		// ok is true if the assertion succeeds (err is of that type)
		if _, ok := err.(*types.ResourceAlreadyExistsException); ok {
			// Someone else created the log group between our check and create attempt
			// This is fine - return success
			return nil
		}
		// Some other error occurred - return it
		return err
	}

	// Set log retention policy if specified in configuration
	// This determines how long AWS keeps the logs before automatically deleting them
	if retentionStr, exists := c.config.Options["retention_days"]; exists {
		// strconv.Atoi converts string to int
		// "Atoi" stands for "ASCII to integer"
		// Multiple assignment captures both the converted value and any error
		if retention, parseErr := strconv.Atoi(retentionStr); parseErr == nil {
			// Only set retention if string parsing succeeded (parseErr == nil)
			// PutRetentionPolicy sets how long CloudWatch keeps the logs
			c.client.PutRetentionPolicy(ctx, &cloudwatchlogs.PutRetentionPolicyInput{
				LogGroupName: aws.String(c.logGroupName),
				// Type conversion from int to int32 since AWS SDK expects int32
				// aws.Int32() converts int32 to *int32 (pointer)
				RetentionInDays: aws.Int32(int32(retention)),
			})
			// Note: We ignore the error from PutRetentionPolicy since it's optional
		}
	}

	// Return nil indicating successful log group creation
	return nil
}

// createLogStream creates a log stream within the log group.
// Log streams organize log events chronologically within a log group.
func (c *CloudWatchDriver) createLogStream() error {
	ctx := context.TODO()

	// Create the log stream
	// CreateLogStream is an AWS CloudWatch Logs API operation
	_, err := c.client.CreateLogStream(ctx, &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(c.logGroupName),  // Which log group to create the stream in
		LogStreamName: aws.String(c.logStreamName), // Name of the new log stream
	})

	// Handle creation errors
	if err != nil {
		// Check if the stream already exists
		// Type assertion to check for specific AWS error type
		if _, ok := err.(*types.ResourceAlreadyExistsException); ok {
			// Stream already exists - this is fine
			return nil
		}
		// Some other error - return it
		return err
	}

	// Stream created successfully
	return nil
}

// putLogEvents sends a batch of log events to CloudWatch Logs.
// This is a private helper method that handles the AWS API interaction.
// events is a slice of InputLogEvent structs containing the log data.
func (c *CloudWatchDriver) putLogEvents(events []types.InputLogEvent) error {
	// Early return pattern - check for empty input
	// len() returns the length of a slice
	if len(events) == 0 {
		return nil // Nothing to send
	}

	ctx := context.TODO()

	// Create the input structure for the AWS API call
	// &cloudwatchlogs.PutLogEventsInput{...} creates struct literal and takes address
	input := &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(c.logGroupName),  // Target log group
		LogStreamName: aws.String(c.logStreamName), // Target log stream
		LogEvents:     events,                      // The actual log events to send
	}

	// Add sequence token if we have one
	// Sequence tokens ensure log events are processed in chronological order
	// nil check: c.sequenceToken is a *string, so we check if it points to something
	if c.sequenceToken != nil {
		// Assign the sequence token to maintain proper ordering
		input.SequenceToken = c.sequenceToken
	}

	// Send the log events to AWS CloudWatch
	// PutLogEvents is the main AWS API call for sending log data
	output, err := c.client.PutLogEvents(ctx, input)
	if err != nil {
		// Error wrapping - add context about what operation failed
		return fmt.Errorf("failed to put log events: %w", err)
	}

	// Update our sequence token for the next batch of events
	// AWS returns a new sequence token that must be used for subsequent calls
	// This ensures proper ordering of log events in CloudWatch
	c.sequenceToken = output.NextSequenceToken

	// Success - no error to return
	return nil
}

// CreateDashboard would create a CloudWatch dashboard for visualizing metrics.
// This is a placeholder method showing how additional AWS functionality could be added.
// Currently unimplemented - returns an error indicating it's not available.
func (c *CloudWatchDriver) CreateDashboard() error {
	// fmt.Errorf creates an error with a formatted message
	// This is Go's way of creating custom error messages
	return fmt.Errorf("dashboard creation not implemented")
}

// CreateAlarms would create CloudWatch alarms for monitoring thresholds.
// This is another placeholder for potential future functionality.
// Alarms could trigger notifications when CPU/memory usage exceeds thresholds.
func (c *CloudWatchDriver) CreateAlarms() error {
	return fmt.Errorf("alarm creation not implemented")
}

/*
Key Go Language Concepts Demonstrated in CloudWatch Driver:

1. **Package System**: Organized code with imports from standard library and external packages
2. **Struct Definition**: CloudWatchDriver struct encapsulates state and behavior
3. **Methods with Receivers**: Functions that operate on struct instances
4. **Interface Implementation**: Implements LogDriver interface methods
5. **Pointers**: Extensive use of pointers (*string, *Client) for memory efficiency
6. **Error Handling**: Explicit error checking and wrapping throughout
7. **Maps**: map[string]interface{} for flexible data structures
8. **Slices**: Dynamic arrays for collecting log events
9. **Type Assertions**: Checking specific error types with comma ok idiom
10. **JSON Marshaling**: Converting Go structs to JSON for CloudWatch
11. **Multiple Return Values**: Functions returning (value, error) pairs
12. **String Conversion**: Converting between string, []byte, and *string types
13. **Time Handling**: Working with timestamps and Unix time
14. **Context Usage**: Passing context for API call management
15. **Constructor Pattern**: NewCloudWatchDriver function for object creation
16. **Private vs Public**: Lowercase vs uppercase method names for visibility
17. **Struct Literals**: Creating and initializing structs with field names
18. **Zero Values**: Understanding nil for pointers and empty slices
19. **Type Conversions**: Converting between int, int32, int64, string types
20. **AWS SDK Patterns**: Using AWS SDK v2 patterns for service clients

AWS CloudWatch Concepts:

1. **Log Groups**: Containers that organize related log streams
2. **Log Streams**: Sequences of log events from the same source
3. **Log Events**: Individual log entries with message and timestamp
4. **Sequence Tokens**: Ensure proper chronological ordering of events
5. **Retention Policies**: Automatic deletion of old logs after specified time
6. **Structured Logging**: Using JSON for searchable, filterable log data
7. **Batch Operations**: Sending multiple log events in single API call
8. **Resource Management**: Creating AWS resources if they don't exist
9. **Error Handling**: Dealing with AWS-specific error types
10. **Configuration**: Using AWS SDK configuration for credentials and region
*/