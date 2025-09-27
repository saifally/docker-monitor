# YAML Configuration Guide

The Docker Monitor now supports **YAML configuration** which is much cleaner and more readable than JSON!

## 🎯 **Why YAML?**

### Before (Ugly JSON):
```bash
LOG_CONFIG='[{"driver":"console","options":{"format":"json","level":"info"}},{"driver":"cloudwatch","options":{"log_group":"/docker-monitor/production","log_stream":"container-monitor","region":"us-east-1","retention_days":"30"}}]'
```

### After (Clean YAML):
```yaml
LOG_CONFIG: |
  - driver: console
    options:
      format: json
      level: info
  - driver: cloudwatch
    options:
      log_group: /docker-monitor/production
      log_stream: container-monitor
      region: us-east-1
      retention_days: "30"
```

## 📝 **Configuration Methods**

### 1. **Environment Variable (YAML)**
```yaml
# docker-compose.yml
environment:
  - LOG_CONFIG=|
    - driver: console
      options:
        format: json
        level: info
```

### 2. **YAML Configuration File**
```yaml
# config.yaml
- driver: console
  options:
    format: json
    level: info
- driver: csv
  output_path: /logs/metrics.csv
```

### 3. **Kubernetes ConfigMap**
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
```

## 🚀 **Quick Examples**

### **Console Only (Kubernetes)**
```yaml
- driver: console
  options:
    format: json
    level: info
```

### **Development Setup**
```yaml
- driver: console
  options:
    format: text
    level: debug
- driver: csv
  output_path: ./logs/dev_metrics.csv
- driver: excel
  output_path: ./logs/dev_metrics.xlsx
  options:
    sheet_name: Development Metrics
```

### **Production Setup**
```yaml
- driver: console
  options:
    format: json
    level: info
- driver: cloudwatch
  options:
    log_group: /k8s/docker-monitor
    log_stream: container-monitor-prod
    region: us-east-1
    retention_days: "90"
```

## 🔧 **Usage Examples**

### **Docker Compose**
```yaml
services:
  docker-monitor:
    environment:
      - LOG_CONFIG=|
        - driver: console
          options:
            format: json
```

### **Kubernetes Deployment**
```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: docker-monitor
        env:
        - name: LOG_CONFIG
          value: |
            - driver: console
              options:
                format: json
                level: info
```

### **Configuration File**
```yaml
# Mount config.yaml as volume
volumes:
  - ./config.yaml:/app/config.yaml:ro
```

## ✨ **Features**

- ✅ **Auto-detection**: Supports both YAML and JSON (backward compatible)
- ✅ **File extension detection**: `.yaml`, `.yml`, `.json`
- ✅ **Environment variables**: Clean YAML in Docker Compose
- ✅ **Configuration files**: External YAML/JSON files
- ✅ **Kubernetes ready**: Perfect for ConfigMaps and environment variables

## 🎯 **Recommendations**

- **Kubernetes**: Use YAML in environment variables or ConfigMaps
- **Development**: Use YAML configuration files
- **Production**: Use YAML with console + CloudWatch drivers
- **Legacy**: JSON still supported for backward compatibility

YAML makes configuration much more maintainable and human-readable! 🎉