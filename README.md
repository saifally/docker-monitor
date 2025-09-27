# Docker Monitor

A lightweight Docker container monitoring tool that tracks resource usage and performance metrics for containerized applications within Docker Compose projects.

## Features

- 🐳 **Docker Compose Integration**: Automatically monitors containers within the same Docker Compose project
- 📊 **Resource Monitoring**: Tracks CPU usage, memory consumption, and container status
- 🔄 **Real-time Metrics**: Configurable monitoring frequency with live updates
- 📝 **Structured Logging**: JSON-formatted logs for easy parsing and analysis
- 🚀 **Lightweight**: Minimal resource footprint using Alpine Linux
- 🛡️ **Secure**: Runs with appropriate Docker socket permissions

## Quick Start

### Using Docker Compose (Recommended)

1. Clone the repository:
```bash
git clone <repository-url>
cd docker-monitor
```

2. Start the monitoring stack:
```bash
docker compose up -d
```

3. View the logs:
```bash
docker compose logs -f docker-monitor
```

### Manual Docker Build

```bash
# Build the image
docker build -t docker-monitor .

# Run the container
docker run -d \
  --name docker-monitor \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --privileged \
  -e MONITOR_FREQUENCY=30s \
  -e COMPOSE_PROJECT_NAME=your-project \
  docker-monitor
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MONITOR_FREQUENCY` | `30s` | How often to collect metrics (e.g., `10s`, `1m`, `5m`) |
| `COMPOSE_PROJECT_NAME` | `default` | Docker Compose project name to monitor |
| `DOCKER_COMPOSE_PROJECT` | - | Alternative project name variable |
| `LOG_CONFIG` | Console only | YAML configuration for logging drivers |
| `CONFIG_FILE` | `/app/config.yaml` | Path to YAML configuration file |

### Logging Configuration

The monitor supports multiple logging drivers that can be configured via YAML format either through environment variables or configuration files.

#### Available Drivers

- **Console**: Outputs logs to stdout/stderr with JSON or text format
- **CSV**: Writes metrics to CSV files with automatic header generation
- **Excel**: Creates Excel workbooks with metrics and formatted data
- **CloudWatch**: Sends logs to AWS CloudWatch Logs with structured JSON

#### YAML Configuration Examples

**Single Driver (Console)**
```yaml
- driver: console
  options:
    format: json
    level: info
```

**Multiple Drivers**
```yaml
- driver: console
  options:
    format: text
    level: info
- driver: csv
  output_path: /logs/metrics.csv
- driver: excel
  output_path: /logs/metrics.xlsx
  options:
    sheet_name: Metrics
```

**CloudWatch Configuration**
```yaml
- driver: cloudwatch
  options:
    log_group: /docker-monitor/production
    log_stream: container-monitor-prod
    region: us-east-1
    retention_days: "30"
```

#### Configuration Methods

**1. Environment Variable (Docker Compose)**
```yaml
services:
  docker-monitor:
    environment:
      LOG_CONFIG: |
        - driver: console
          options:
            format: json
            level: info
        - driver: csv
          output_path: /logs/metrics.csv
```

**2. Configuration File**
```yaml
# config.yaml
- driver: console
  options:
    format: json
    level: info
- driver: csv
  output_path: /logs/container_metrics.csv
  options:
    info_log_file: /logs/info.log
    error_log_file: /logs/errors.log
```

**3. Kubernetes ConfigMap**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: docker-monitor-config
data:
  config.yaml: |
    - driver: console
      options:
        format: json
        level: info
    - driver: cloudwatch
      options:
        log_group: /k8s/docker-monitor
        log_stream: container-monitor
        region: us-east-1
```

### Docker Compose Configuration

The included `docker-compose.yml` provides a complete monitoring setup with YAML logging configuration:

```yaml
version: '3.8'

services:
  docker-monitor:
    build: .
    container_name: docker-monitor
    environment:
      MONITOR_FREQUENCY: 30s
      COMPOSE_PROJECT_NAME: ${COMPOSE_PROJECT_NAME:-docker-monitor}
      # Clean YAML configuration using multi-line string
      LOG_CONFIG: |
        - driver: console
          options:
            format: json
            level: info
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./logs:/logs  # For CSV/Excel output
      - ./config.yaml:/app/config.yaml:ro  # YAML config file
    restart: unless-stopped
    depends_on:
      - app
      - redis
    networks:
      - monitor-network

  # Example services to monitor
  app:
    image: nginx:alpine
    container_name: example-app
    ports:
      - "8081:80"
    networks:
      - monitor-network

  redis:
    image: redis:alpine
    networks:
      - monitor-network

networks:
  monitor-network:
    driver: bridge
```

## Output Format

The monitor outputs structured JSON logs with container metrics:

### Container Report Log
```json
{
  "project": "docker-monitor",
  "containers_count": 4,
  "monitoring_frequency": "30s",
  "level": "info",
  "msg": "Container monitoring report",
  "time": "2023-12-01T10:30:00Z"
}
```

### Individual Container Metrics
```json
{
  "container_name": "example-app",
  "status": "Up 2 minutes",
  "image": "nginx:alpine",
  "cpu_percent": 0.5,
  "memory_usage": "7.4 MB",
  "memory_limit": "15.5 GB",
  "compose_service": "app",
  "level": "info",
  "msg": "Container metrics",
  "time": "2023-12-01T10:30:00Z"
}
```

## Project Structure

```
docker-monitor/
├── main.go              # Main application code
├── main_test.go         # Unit tests
├── go.mod               # Go module dependencies
├── logger/              # Logging drivers package
│   ├── types.go         # Common types and interfaces
│   ├── manager.go       # Log manager orchestration
│   ├── console.go       # Console logging driver
│   ├── csv.go          # CSV file logging driver
│   ├── excel.go        # Excel file logging driver
│   └── cloudwatch.go   # AWS CloudWatch logging driver
├── config.yaml          # Default YAML configuration
├── examples/            # Configuration examples
│   ├── console-only.yaml
│   ├── development.yaml
│   ├── kubernetes.yaml
│   └── console-cloudwatch.yaml
├── Dockerfile           # Container build configuration
├── docker-compose.yml   # Complete monitoring stack with YAML config
└── README.md           # This file
```

## Development

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- Access to Docker socket

### Running Tests

```bash
# Run all tests
go test -v

# Run with coverage
go test -v -cover

# Run benchmarks
go test -bench=.
```

### Building from Source

```bash
# Install dependencies
go mod tidy

# Build binary
go build -o docker-monitor .

# Run locally (requires Docker socket access)
./docker-monitor
```

## Monitoring Logic

The monitor identifies containers belonging to the same Docker Compose project using several label patterns:

1. `com.docker.compose.project`
2. `com.docker.compose.project.name`
3. `docker-compose.project`
4. Service name prefix matching (`com.docker.compose.service`)

### CPU Calculation

CPU percentage is calculated using the formula:
```
CPU% = (CPU_delta / System_delta) × CPU_count × 100
```

Where:
- `CPU_delta` = Current CPU usage - Previous CPU usage
- `System_delta` = Current system usage - Previous system usage
- `CPU_count` = Number of CPU cores available to the container

### Memory Formatting

Memory values are automatically formatted with appropriate units (B, KB, MB, GB, TB, PB, EB) using 1024-byte increments.

## Troubleshooting

### Common Issues

#### Permission Denied Error
```
permission denied while trying to connect to the Docker daemon socket
```

**Solution**: Ensure the container runs with `privileged: true` or add the user to the docker group:
```yaml
privileged: true
```

#### No Containers Found
```
Container monitoring report: containers_count=0
```

**Solutions**:
1. Check that `COMPOSE_PROJECT_NAME` matches your actual project name
2. Verify containers have proper Docker Compose labels
3. Ensure containers are running in the same Docker network

#### High Resource Usage
If the monitor consumes too many resources:
1. Increase `MONITOR_FREQUENCY` (e.g., `60s` instead of `30s`)
2. Reduce the number of monitored containers
3. Check for Docker API connectivity issues

### Docker Desktop on Windows

When using Docker Desktop on Windows, you may need:
1. Enable "Expose daemon on tcp://localhost:2375 without TLS" in Docker Desktop settings
2. Use `privileged: true` in docker-compose.yml
3. Ensure Docker Desktop is running with administrator privileges

### Logs and Debugging

View detailed logs:
```bash
# Follow logs in real-time
docker compose logs -f docker-monitor

# View last 50 lines
docker compose logs --tail=50 docker-monitor

# Check container status
docker compose ps
```

## Performance

The monitor is designed for minimal overhead:

- **CPU Usage**: < 0.1% on average
- **Memory Usage**: ~3-5 MB RAM
- **Network**: Minimal (local Docker API calls only)
- **Disk**: No persistent storage required

Benchmark results:
- `formatBytes`: ~290 ns/op
- `isFromSameComposeProject`: ~5.6 ns/op

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass (`go test -v`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

## License

This project is open source and available under the [MIT License](LICENSE).

## Support

- Create an issue for bug reports or feature requests
- Check existing issues before creating new ones
- Provide Docker version, OS, and error logs when reporting issues

---

**Note**: This tool requires access to the Docker daemon socket and should be used with appropriate security considerations in production environments.