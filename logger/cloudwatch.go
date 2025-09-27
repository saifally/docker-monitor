package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type CloudWatchDriver struct {
	config       LogConfig
	client       *cloudwatchlogs.Client
	logGroupName string
	logStreamName string
	sequenceToken *string
}

func NewCloudWatchDriver(config LogConfig) *CloudWatchDriver {
	logGroupName := "/docker-monitor/container-metrics"
	if group, exists := config.Options["log_group"]; exists {
		logGroupName = group
	}

	logStreamName := fmt.Sprintf("container-monitor-%d", time.Now().Unix())
	if stream, exists := config.Options["log_stream"]; exists {
		logStreamName = stream
	}

	return &CloudWatchDriver{
		config:        config,
		logGroupName:  logGroupName,
		logStreamName: logStreamName,
	}
}

func (c *CloudWatchDriver) Initialize() error {
	// Load AWS config
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Override region if specified
	if region, exists := c.config.Options["region"]; exists {
		cfg.Region = region
	}

	c.client = cloudwatchlogs.NewFromConfig(cfg)

	// Create log group if it doesn't exist
	if err := c.createLogGroupIfNotExists(); err != nil {
		return fmt.Errorf("failed to create log group: %w", err)
	}

	// Create log stream
	if err := c.createLogStream(); err != nil {
		return fmt.Errorf("failed to create log stream: %w", err)
	}

	return nil
}

func (c *CloudWatchDriver) LogMetrics(metrics []ContainerMetrics) error {
	events := make([]types.InputLogEvent, 0, len(metrics)+1)

	// Add summary event
	summaryData := map[string]interface{}{
		"event_type":       "monitoring_summary",
		"containers_count": len(metrics),
		"timestamp":        time.Now(),
	}

	summaryJSON, _ := json.Marshal(summaryData)
	events = append(events, types.InputLogEvent{
		Message:   aws.String(string(summaryJSON)),
		Timestamp: aws.Int64(time.Now().UnixMilli()),
	})

	// Add individual metric events
	for _, metric := range metrics {
		logData := map[string]interface{}{
			"event_type":     "container_metrics",
			"container_name": metric.Name,
			"status":         metric.Status,
			"image":          metric.Image,
			"cpu_percent":    metric.CPUPercent,
			"memory_usage":   metric.MemoryUsage,
			"memory_limit":   metric.MemoryLimit,
			"project":        metric.Project,
			"timestamp":      metric.Timestamp,
		}

		// Add compose service if available
		if service, exists := metric.Labels["com.docker.compose.service"]; exists {
			logData["compose_service"] = service
		}

		// Add other useful labels
		for key, value := range metric.Labels {
			if key == "com.docker.compose.version" ||
			   key == "com.docker.compose.config-hash" ||
			   key == "com.docker.compose.container-number" {
				logData[key] = value
			}
		}

		logJSON, err := json.Marshal(logData)
		if err != nil {
			continue // Skip malformed entries
		}

		events = append(events, types.InputLogEvent{
			Message:   aws.String(string(logJSON)),
			Timestamp: aws.Int64(metric.Timestamp.UnixMilli()),
		})
	}

	// Send to CloudWatch
	return c.putLogEvents(events)
}

func (c *CloudWatchDriver) LogInfo(message string, fields map[string]interface{}) error {
	logData := map[string]interface{}{
		"event_type": "info",
		"message":    message,
		"timestamp":  time.Now(),
	}

	for k, v := range fields {
		logData[k] = v
	}

	logJSON, err := json.Marshal(logData)
	if err != nil {
		return err
	}

	events := []types.InputLogEvent{
		{
			Message:   aws.String(string(logJSON)),
			Timestamp: aws.Int64(time.Now().UnixMilli()),
		},
	}

	return c.putLogEvents(events)
}

func (c *CloudWatchDriver) LogError(message string, err error) error {
	logData := map[string]interface{}{
		"event_type": "error",
		"message":    message,
		"error":      err.Error(),
		"timestamp":  time.Now(),
	}

	logJSON, jsonErr := json.Marshal(logData)
	if jsonErr != nil {
		return jsonErr
	}

	events := []types.InputLogEvent{
		{
			Message:   aws.String(string(logJSON)),
			Timestamp: aws.Int64(time.Now().UnixMilli()),
		},
	}

	return c.putLogEvents(events)
}

func (c *CloudWatchDriver) Close() error {
	// Send final close event
	logData := map[string]interface{}{
		"event_type": "monitor_shutdown",
		"message":    "Docker monitor shutting down",
		"timestamp":  time.Now(),
	}

	logJSON, _ := json.Marshal(logData)
	events := []types.InputLogEvent{
		{
			Message:   aws.String(string(logJSON)),
			Timestamp: aws.Int64(time.Now().UnixMilli()),
		},
	}

	c.putLogEvents(events)
	return nil
}

func (c *CloudWatchDriver) createLogGroupIfNotExists() error {
	ctx := context.TODO()

	// Check if log group exists
	_, err := c.client.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: aws.String(c.logGroupName),
	})

	if err == nil {
		return nil // Log group exists
	}

	// Create log group
	_, err = c.client.CreateLogGroup(ctx, &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(c.logGroupName),
	})

	if err != nil {
		// Check if error is because group already exists
		if _, ok := err.(*types.ResourceAlreadyExistsException); ok {
			return nil
		}
		return err
	}

	// Set retention policy if specified
	if retentionStr, exists := c.config.Options["retention_days"]; exists {
		if retention, parseErr := strconv.Atoi(retentionStr); parseErr == nil {
			c.client.PutRetentionPolicy(ctx, &cloudwatchlogs.PutRetentionPolicyInput{
				LogGroupName:    aws.String(c.logGroupName),
				RetentionInDays: aws.Int32(int32(retention)),
			})
		}
	}

	return nil
}

func (c *CloudWatchDriver) createLogStream() error {
	ctx := context.TODO()

	_, err := c.client.CreateLogStream(ctx, &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(c.logGroupName),
		LogStreamName: aws.String(c.logStreamName),
	})

	if err != nil {
		// Check if error is because stream already exists
		if _, ok := err.(*types.ResourceAlreadyExistsException); ok {
			return nil
		}
		return err
	}

	return nil
}

func (c *CloudWatchDriver) putLogEvents(events []types.InputLogEvent) error {
	if len(events) == 0 {
		return nil
	}

	ctx := context.TODO()

	input := &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(c.logGroupName),
		LogStreamName: aws.String(c.logStreamName),
		LogEvents:     events,
	}

	if c.sequenceToken != nil {
		input.SequenceToken = c.sequenceToken
	}

	output, err := c.client.PutLogEvents(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put log events: %w", err)
	}

	// Update sequence token for next request
	c.sequenceToken = output.NextSequenceToken
	return nil
}

// CreateDashboard creates a CloudWatch dashboard for monitoring
func (c *CloudWatchDriver) CreateDashboard() error {
	// This would create a CloudWatch dashboard to visualize the metrics
	// Implementation would depend on specific requirements
	return fmt.Errorf("dashboard creation not implemented")
}

// CreateAlarms creates CloudWatch alarms for critical metrics
func (c *CloudWatchDriver) CreateAlarms() error {
	// This would create CloudWatch alarms for high CPU/memory usage
	// Implementation would depend on specific requirements
	return fmt.Errorf("alarm creation not implemented")
}