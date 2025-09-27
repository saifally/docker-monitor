# Docker Monitor Architecture

This folder contains the architecture diagrams for the Docker Monitor application, generated using the [Diagrams](https://diagrams.mingrammer.com/) library (mingrammer).

## Architecture Overview

The Docker Monitor application is a containerized Go application that monitors Docker containers and logs metrics to multiple output destinations.

### Key Components

1. **Main Monitor Application** (`main.go`)
   - `Monitor` struct: Core monitoring logic
   - Constructor pattern for initialization
   - CPU and memory calculation functions
   - Project-based container filtering

2. **Logger Package** (`logger/`)
   - `LogManager`: Orchestrates multiple logging drivers
   - `LogDriver` interface: Defines logging contract
   - Multiple driver implementations:
     - Console Driver (stdout/stderr)
     - CSV Driver (file output)
     - Excel Driver (XLSX files)
     - CloudWatch Driver (AWS logging)

3. **Configuration**
   - Environment variables
   - YAML configuration files
   - Docker Compose setup

4. **Docker Integration**
   - Docker API client
   - Container statistics collection
   - Real-time monitoring with configurable frequency

## Generating the Diagram

### Prerequisites

1. **Install Python dependencies:**
   ```bash
   pip install -r requirements.txt
   ```

2. **Install Graphviz:**
   - **Windows:** Download from [graphviz.org](https://graphviz.org/download/)
   - **macOS:** `brew install graphviz`
   - **Linux:** `apt-get install graphviz` or `yum install graphviz`

### Generate Diagram

```bash
cd architecture
python generate_diagram.py
```

This will create `docker_monitor_architecture.png` in the current directory.

## Architecture Pattern

The application follows several design patterns:

- **Manager/Orchestrator Pattern**: LogManager coordinates multiple drivers
- **Strategy Pattern**: Different logging strategies via LogDriver implementations
- **Factory Pattern**: Driver creation based on configuration
- **Interface Segregation**: Clean separation between components
- **Composition over Inheritance**: Go's approach to code reuse

## Data Flow

1. **Initialization**: Read configuration from environment variables or YAML files
2. **Docker Connection**: Establish connection to Docker daemon via API
3. **Container Discovery**: List and filter containers by project
4. **Metrics Collection**: Gather CPU, memory, and status metrics
5. **Logging**: Route metrics to configured output destinations
6. **Continuous Monitoring**: Repeat at configurable intervals

## Output Destinations

- **Console**: JSON formatted logs to stdout/stderr
- **CSV Files**: Structured data for analysis
- **Excel Files**: Business-friendly reports
- **AWS CloudWatch**: Cloud-based logging and monitoring