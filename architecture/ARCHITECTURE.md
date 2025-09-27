# Docker Monitor Architecture Documentation

## System Overview

The Docker Monitor is a containerized Go application that provides real-time monitoring of Docker containers with multiple logging output destinations. It follows modern software architecture patterns including the Manager/Orchestrator pattern, Strategy pattern, and Interface segregation.

## Architecture Diagram

```
                           Administrator
                                |
                                |
                    ┌─────────────────────────┐
                    │   Configuration Sources │
                    │  ┌─────────────────────┐ │
                    │  │ Environment Variables│ │
                    │  │ config.yaml (YAML)  │ │
                    │  └─────────────────────┘ │
                    └─────────────────────────┘
                                |
                                ↓
              ┌───────────────────────────────────────────┐
              │         Docker Monitor Application        │
              │                                           │
              │  ┌─────────────────────────────────────┐  │
              │  │        Main Process (main.go)       │  │
              │  │  ┌───────────┐ ┌──────────────────┐ │  │
              │  │  │  Monitor  │ │ NewMonitor()     │ │  │
              │  │  │  Struct   │ │ Constructor      │ │  │
              │  │  └───────────┘ └──────────────────┘ │  │
              │  │  ┌───────────────────────────────┐ │  │
              │  │  │  CPU & Memory Calculations   │ │  │
              │  │  └───────────────────────────────┘ │  │
              │  └─────────────────────────────────────┘  │
              │                     │                     │
              │  ┌─────────────────────────────────────┐  │
              │  │         Logger Package              │  │
              │  │  ┌───────────┐ ┌──────────────────┐ │  │
              │  │  │LogManager │ │ LogDriver        │ │  │
              │  │  │Orchestrator│ │ Interface        │ │  │
              │  │  └───────────┘ └──────────────────┘ │  │
              │  └─────────────────────────────────────┘  │
              │                     │                     │
              │  ┌─────────────────────────────────────┐  │
              │  │          Logging Drivers            │  │
              │  │ ┌─────────┐ ┌─────────┐ ┌─────────┐ │  │
              │  │ │Console  │ │   CSV   │ │ Excel   │ │  │
              │  │ │ Driver  │ │ Driver  │ │ Driver  │ │  │
              │  │ └─────────┘ └─────────┘ └─────────┘ │  │
              │  │ ┌─────────────────────────────────┐ │  │
              │  │ │      CloudWatch Driver          │ │  │
              │  │ └─────────────────────────────────┘ │  │
              │  └─────────────────────────────────────┘  │
              └───────────────────────────────────────────┘
                                |
                                ↓
                ┌─────────────────────────────┐
                │      Docker Environment     │
                │  ┌─────────┐ ┌────────────┐ │
                │  │ Docker  │ │   Docker   │ │
                │  │ Daemon  │ │    API     │ │
                │  └─────────┘ └────────────┘ │
                │  ┌─────────────────────────┐ │
                │  │      Containers         │ │
                │  │ ┌─────┐ ┌─────┐ ┌─────┐ │ │
                │  │ │ App │ │Redis│ │ DB  │ │ │
                │  │ └─────┘ └─────┘ └─────┘ │ │
                │  └─────────────────────────┘ │
                └─────────────────────────────┘
                                |
                                ↓
              ┌───────────────────────────────────────────┐
              │          Output Destinations              │
              │ ┌─────────┐ ┌─────────┐ ┌─────────────┐  │
              │ │Console  │ │CSV Files│ │ Excel Files │  │
              │ │stdout/  │ │/logs/   │ │ /logs/*.xlsx│  │
              │ │stderr   │ │*.csv    │ │             │  │
              │ └─────────┘ └─────────┘ └─────────────┘  │
              │ ┌─────────────────────────────────────┐  │
              │ │         AWS CloudWatch Logs        │  │
              │ └─────────────────────────────────────┘  │
              └───────────────────────────────────────────┘
```

## Core Components

### 1. Main Application (main.go)

**Monitor Struct**: Central orchestrator containing:
- `client`: Docker API client connection
- `logManager`: Logging orchestrator instance
- `frequency`: Monitoring interval (default: 30s)
- `project`: Docker Compose project filter

**Key Functions**:
- `NewMonitor()`: Constructor with environment/YAML configuration
- `getContainerStats()`: Retrieves container metrics via Docker API
- `calculateCPUPercent()`: Computes CPU usage percentage
- `formatBytes()`: Human-readable memory formatting (B/KB/MB/GB/TB)
- `isFromSameComposeProject()`: Filters containers by project labels
- `monitorContainers()`: Main monitoring loop
- `Start()`: Application entry point with ticker-based scheduling

### 2. Logger Package (logger/)

**LogManager**: Implements Manager/Orchestrator pattern
- Coordinates multiple logging drivers simultaneously
- Fan-out pattern: one input → multiple outputs
- YAML/environment configuration loading
- Error aggregation across drivers

**LogDriver Interface**: Strategy pattern implementation
```go
type LogDriver interface {
    Initialize() error
    LogMetrics(metrics []ContainerMetrics) error
    LogInfo(message string, fields map[string]interface{}) error
    LogError(message string, err error) error
    Close() error
}
```

**Driver Implementations**:
- **Console Driver**: JSON formatted logs to stdout/stderr
- **CSV Driver**: Structured data files with separate info/error logs
- **Excel Driver**: Business-friendly XLSX reports with worksheets
- **CloudWatch Driver**: AWS cloud logging integration

### 3. Configuration System

**Environment Variables**:
- `MONITOR_FREQUENCY`: Monitoring interval (30s, 1m, 5m, etc.)
- `COMPOSE_PROJECT_NAME`: Primary project filter
- `DOCKER_COMPOSE_PROJECT`: Fallback project filter
- `CONFIG_FILE`: YAML configuration file path
- `LOG_CONFIG`: Direct YAML configuration string

**YAML Configuration**:
```yaml
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

### 4. Data Flow

1. **Initialization Phase**:
   - Read configuration from environment/YAML
   - Initialize Docker API client
   - Create LogManager with configured drivers
   - Set up monitoring frequency and project filters

2. **Monitoring Loop**:
   - List all containers via Docker API
   - Filter containers by Compose project labels
   - Collect real-time statistics for running containers
   - Calculate CPU percentage and format memory usage
   - Create ContainerMetrics structs with metadata

3. **Logging Phase**:
   - LogManager receives metrics from Monitor
   - Fan-out to all configured drivers simultaneously
   - Each driver processes metrics according to its strategy
   - Error handling with fallback to console output

## Design Patterns

### Manager/Orchestrator Pattern
`LogManager` coordinates multiple `LogDriver` implementations, providing centralized control while maintaining loose coupling.

### Strategy Pattern
Different logging strategies (`ConsoleDriver`, `CSVDriver`, etc.) implement the same `LogDriver` interface, allowing runtime driver selection.

### Factory Pattern
Driver creation based on configuration type in `LogManager.AddDriver()`.

### Fan-Out Pattern
Single metrics input distributed to multiple output destinations simultaneously.

### Interface Segregation
Clean separation between Docker operations, logging operations, and configuration management.

## Container Metrics Schema

```go
type ContainerMetrics struct {
    Name        string            // Container name (without /)
    Status      string            // Container status (Up, Exited, etc.)
    Image       string            // Docker image name
    CPUPercent  float64           // CPU usage percentage
    MemoryUsage string            // Formatted memory usage (e.g., "256 MB")
    MemoryLimit string            // Formatted memory limit
    Timestamp   time.Time         // Collection timestamp
    Labels      map[string]string // All container labels
    Project     string            // Compose project name
}
```

## Docker Integration

### Container Discovery
- Uses Docker API `ContainerList()` with `All: true`
- Filters by Docker Compose labels:
  - `com.docker.compose.project`
  - `com.docker.compose.project.name`
  - `docker-compose.project`
  - `com.docker.compose.service` (prefix matching)

### Statistics Collection
- Real-time container stats via Docker API `ContainerStats()`
- JSON decoding of statistics stream
- CPU percentage calculation using delta method
- Memory usage and limit extraction

### API Error Handling
- Graceful degradation when containers are unreachable
- Individual container failures don't stop monitoring
- Comprehensive error logging with context

## Deployment Architecture

### Docker Compose Integration
```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock:ro  # Docker API access
  - ./logs:/logs                                  # Output directory
  - ./config.yaml:/app/config.yaml:ro            # Configuration
```

### Security Considerations
- Read-only Docker socket mount
- No privileged container access required
- Environment-based configuration for secrets
- AWS IAM roles for CloudWatch access

## Performance Characteristics

### Resource Usage
- Minimal CPU overhead (Go's efficient concurrency)
- Low memory footprint (typically <50MB)
- Configurable monitoring frequency for resource tuning

### Scalability
- Handles dozens of containers efficiently
- Logarithmic complexity for container discovery
- Parallel logging to multiple destinations

### Error Resilience
- Continue monitoring despite individual container failures
- Automatic reconnection to Docker daemon
- Graceful shutdown with resource cleanup

## Output Formats

### Console (JSON)
```json
{
  "level": "info",
  "msg": "Container metrics collected",
  "containers": 5,
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### CSV
```csv
Name,Status,Image,CPU %,Memory Usage,Memory Limit,Timestamp,Project
web-1,Up,nginx:latest,2.5,64 MB,512 MB,2024-01-01T12:00:00Z,myapp
```

### Excel
Structured workbook with formatted columns, charts, and summary sheets.

### CloudWatch
Structured logs with searchable fields and CloudWatch Insights compatibility.

## Development & Testing

### Unit Testing
- Comprehensive test coverage (>90%)
- Mock implementations for Docker client and LogManager
- Environment variable testing with setup/teardown
- Edge case validation for calculations and formatting

### CI/CD Integration
- GitHub Actions workflow for build and test
- Cross-platform compatibility (Linux, macOS, Windows)
- Docker image building and publishing

## Future Enhancements

### Planned Features
- Prometheus metrics export
- Grafana dashboard templates
- Health check endpoints
- Configuration hot-reloading
- Multi-project monitoring
- Alert threshold configuration

### Extensibility Points
- Additional LogDriver implementations
- Custom metric calculations
- Plugin architecture for external integrations
- REST API for runtime management

---

This architecture provides a robust, scalable, and maintainable solution for Docker container monitoring with flexible output options and comprehensive error handling.