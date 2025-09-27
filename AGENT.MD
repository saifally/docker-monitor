# Docker Container Monitor Agent

## Overview

A Go-based monitoring agent that runs inside a Docker container to monitor other containers within the same Docker Compose stack. The agent collects and logs container metrics including names, status, memory usage, CPU usage, and images at configurable intervals.

## Features

- **Compose-aware monitoring**: Only monitors containers from the same Docker Compose project
- **Comprehensive metrics**: Tracks container names, status, images, CPU usage, and memory usage
- **Configurable frequency**: Adjustable monitoring intervals via environment variables
- **Multiple logging drivers**: Support for console, CSV, Excel, and CloudWatch logging
- **Structured logging**: JSON-based logging for easy parsing and analysis
- **Security-focused**: Runs as non-root user with read-only Docker socket access
- **Containerized deployment**: Fully containerized with multi-stage Docker build

## Requirements

### System Requirements
- Docker Engine
- Docker Compose
- Access to Docker socket (`/var/run/docker.sock`)

### Go Dependencies
- `github.com/docker/docker` - Docker client library
- `github.com/docker/go-connections` - Docker connection utilities
- `github.com/sirupsen/logrus` - Structured logging
- `github.com/xuri/excelize/v2` - Excel file generation
- `github.com/aws/aws-sdk-go-v2` - AWS CloudWatch integration

## Configuration

### Environment Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `MONITOR_FREQUENCY` | Monitoring interval | `30s` | `10s`, `1m`, `5m` |
| `COMPOSE_PROJECT_NAME` | Docker Compose project name | `default` | `myapp` |
| `LOG_CONFIG` | JSON configuration for logging drivers | Console only | See Logging Configuration |

### Logging Configuration

The agent supports multiple logging drivers that can be configured via the `LOG_CONFIG` environment variable or a configuration file. Supported drivers:

#### Console Driver
- **Description**: Outputs logs to stdout/stderr
- **Format**: JSON or text format
- **Use case**: Development, debugging, container logs

#### CSV Driver
- **Description**: Writes metrics to CSV files
- **Features**: Automatic header generation, file rotation support
- **Use case**: Data analysis, spreadsheet import, long-term storage

#### Excel Driver
- **Description**: Creates Excel workbooks with metrics
- **Features**: Multiple sheets, charts, formatted data
- **Use case**: Reporting, business intelligence, data visualization

#### CloudWatch Driver
- **Description**: Sends logs to AWS CloudWatch Logs
- **Features**: Structured JSON logs, retention policies, real-time monitoring
- **Use case**: Production monitoring, alerting, centralized logging

### Example Logging Configurations

#### Single Driver (Console)
```bash
# Environment variable
export LOG_CONFIG='[{"driver":"console","options":{"format":"json","level":"info"}}]'
```

#### Multiple Drivers
```bash
# Environment variable for multiple outputs
export LOG_CONFIG='[
  {"driver":"console","options":{"format":"text","level":"info"}},
  {"driver":"csv","output_path":"/logs/metrics.csv"},
  {"driver":"excel","output_path":"/logs/metrics.xlsx","options":{"sheet_name":"Metrics"}}
]'
```

#### CloudWatch Configuration
```bash
export LOG_CONFIG='[{
  "driver":"cloudwatch",
  "options":{
    "log_group":"/docker-monitor/production",
    "log_stream":"container-monitor-prod",
    "region":"us-east-1",
    "retention_days":"30"
  }
}]'
```

#### Configuration File
Alternatively, create a `config.json` file:
```json
[
  {
    "driver": "console",
    "options": {
      "format": "json",
      "level": "info"
    }
  },
  {
    "driver": "csv",
    "output_path": "/logs/container_metrics.csv",
    "options": {
      "info_log_file": "/logs/info.log",
      "error_log_file": "/logs/errors.log"
    }
  },
  {
    "driver": "excel",
    "output_path": "/logs/container_metrics.xlsx",
    "options": {
      "sheet_name": "Container Metrics"
    }
  }
]
```

### Docker Compose Labels

The agent identifies containers from the same Docker Compose project using these labels:
- `com.docker.compose.project`
- `com.docker.compose.project.name`
- `docker-compose.project`

## Installation & Usage

### 1. Quick Start with Docker Compose

```bash
# Clone or download the project files
git clone <repository-url>
cd docker-monitor

# Start the monitoring stack
docker-compose up -d

# View logs
docker-compose logs -f docker-monitor
```

### 2. Manual Build and Run

```bash
# Build the Docker image
docker build -t docker-monitor .

# Run the container
docker run -d \
  --name docker-monitor \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -e MONITOR_FREQUENCY=30s \
  -e COMPOSE_PROJECT_NAME=myproject \
  docker-monitor
```

### 3. Integration with Existing Docker Compose

Add the monitoring service to your existing `docker-compose.yml`:

```yaml
services:
  docker-monitor:
    build:
      context: ./path/to/docker-monitor
    environment:
      - MONITOR_FREQUENCY=30s
      - COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME}
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    restart: unless-stopped

  # Your existing services...
  your-app:
    image: your-app:latest
    # ... other configuration
```

## Metrics Collected

### Container Information
- **Name**: Container name (without leading slash)
- **Status**: Current container status (running, stopped, etc.)
- **Image**: Docker image used by the container
- **Labels**: All container labels for identification

### Performance Metrics (Running containers only)
- **CPU Percentage**: Real-time CPU usage percentage
- **Memory Usage**: Current memory consumption (formatted)
- **Memory Limit**: Memory limit set for the container (formatted)

### Metadata
- **Timestamp**: When the metrics were collected
- **Compose Service**: Docker Compose service name (if available)

## Log Output Format

The agent outputs JSON-formatted logs for each monitoring cycle:

```json
{
  "level": "info",
  "msg": "Container monitoring report",
  "project": "myproject",
  "containers_count": 3,
  "monitoring_frequency": "30s",
  "time": "2023-12-07T10:30:00Z"
}

{
  "level": "info",
  "msg": "Container metrics",
  "container_name": "myproject_app_1",
  "status": "Up 2 hours",
  "image": "nginx:alpine",
  "cpu_percent": 0.15,
  "memory_usage": "12.3 MB",
  "memory_limit": "512.0 MB",
  "compose_service": "app",
  "time": "2023-12-07T10:30:00Z"
}
```

## Security Considerations

- **Read-only Docker socket**: The container mounts Docker socket in read-only mode
- **Non-root execution**: Runs as user `monitor` (UID 1001) for security
- **Minimal attack surface**: Alpine-based image with only necessary dependencies
- **No network exposure**: Agent doesn't expose any ports externally

## Troubleshooting

### Common Issues

1. **Permission denied accessing Docker socket**
   ```bash
   # Ensure the Docker socket is accessible
   ls -la /var/run/docker.sock
   # On some systems, you may need to add the user to docker group
   ```

2. **No containers detected**
   - Verify `COMPOSE_PROJECT_NAME` matches your Docker Compose project
   - Check that containers have proper Docker Compose labels

3. **High resource usage**
   - Increase `MONITOR_FREQUENCY` to reduce monitoring frequency
   - Monitor the agent's own resource consumption

### Debugging

Enable debug logging by modifying the log level in `main.go`:

```go
logger.SetLevel(logrus.DebugLevel)
```

View agent logs:
```bash
docker-compose logs -f docker-monitor
```

## Performance Impact

- **CPU Usage**: Minimal (~0.1% on modern systems)
- **Memory Usage**: ~10-20MB base consumption
- **Network Impact**: Local Docker API calls only
- **Storage Impact**: Log files grow based on number of containers and frequency

## Customization

### Monitoring Different Metrics

To add custom metrics, modify the `ContainerMetrics` struct and the monitoring logic in `main.go`:

```go
type ContainerMetrics struct {
    // Existing fields...
    NetworkIO   string `json:"network_io"`
    DiskUsage   string `json:"disk_usage"`
}
```

### Different Output Formats

Replace the JSON logger with different formatters:

```go
// For human-readable output
logger.SetFormatter(&logrus.TextFormatter{})

// For custom formats
logger.SetFormatter(&MyCustomFormatter{})
```

## License

[Specify your license here]

## Contributing

[Add contribution guidelines here]