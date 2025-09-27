# Docker Monitor Logging Architecture Diagram

## Class and Interface Relationships

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           MAIN APPLICATION                                 │
│  ┌─────────────────┐                                                      │
│  │     main.go     │                                                      │
│  │                 │                                                      │
│  │  ┌───────────┐  │                                                      │
│  │  │ Monitor   │  │                                                      │
│  │  │ struct    │  │                                                      │
│  │  │           │  │                                                      │
│  │  │ - client  │  │                                                      │
│  │  │ - logMgr  │──┼─────┐                                               │
│  │  │ - freq    │  │     │                                               │
│  │  │ - project │  │     │                                               │
│  │  └───────────┘  │     │                                               │
│  └─────────────────┘     │                                               │
└───────────────────────────┼───────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        LOGGER PACKAGE                                     │
│                                                                           │
│  ┌─────────────────┐    ┌─────────────────────────────────────────────┐  │
│  │   types.go      │    │              manager.go                     │  │
│  │                 │    │                                             │  │
│  │ ┌─────────────┐ │    │  ┌─────────────────────────────────────────┐│  │
│  │ │ LogDriver   │ │    │  │           LogManager                    ││  │
│  │ │ interface   │ │    │  │                                         ││  │
│  │ │             │ │    │  │  - drivers: []LogDriver                 ││  │
│  │ │ Initialize()│ │◄───┼──┤  - configs: []LogConfig                 ││  │
│  │ │ LogMetrics()│ │    │  │                                         ││  │
│  │ │ LogInfo()   │ │    │  │  + LoadConfigFromEnv()                 ││  │
│  │ │ LogError()  │ │    │  │  + LoadConfigFromFile()                ││  │
│  │ │ Close()     │ │    │  │  + AddDriver()                         ││  │
│  │ └─────────────┘ │    │  │  + LogMetrics()                        ││  │
│  │                 │    │  │  + LogInfo()                           ││  │
│  │ ┌─────────────┐ │    │  │  + LogError()                          ││  │
│  │ │ContainerMet-│ │    │  │  + Close()                             ││  │
│  │ │rics struct  │ │    │  └─────────────────────────────────────────┘│  │
│  │ │             │ │    └─────────────────────────────────────────────┘  │
│  │ │ - Name      │ │                                                     │
│  │ │ - Status    │ │                                                     │
│  │ │ - Image     │ │                                                     │
│  │ │ - CPUPercent│ │                                                     │
│  │ │ - Memory... │ │                                                     │
│  │ │ - Timestamp │ │                                                     │
│  │ │ - Labels    │ │                                                     │
│  │ │ - Project   │ │                                                     │
│  │ └─────────────┘ │                                                     │
│  │                 │                                                     │
│  │ ┌─────────────┐ │                                                     │
│  │ │ LogConfig   │ │                                                     │
│  │ │ struct      │ │                                                     │
│  │ │             │ │                                                     │
│  │ │ - Driver    │ │                                                     │
│  │ │ - OutputPath│ │                                                     │
│  │ │ - Options   │ │                                                     │
│  │ └─────────────┘ │                                                     │
│  └─────────────────┘                                                     │
└─────────────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    CONCRETE DRIVER IMPLEMENTATIONS                        │
│                                                                           │
│ ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌────────┐│
│ │  console.go     │  │    csv.go       │  │   excel.go      │  │cloudwa-││
│ │                 │  │                 │  │                 │  │tch.go  ││
│ │┌───────────────┐│  │┌───────────────┐│  │┌───────────────┐│  │┌──────┐││
│ ││ConsoleDriver  ││  ││   CSVDriver   ││  ││  ExcelDriver  ││  ││Cloud-│││
│ ││               ││  ││               ││  ││               ││  ││Watch │││
│ ││- logger       ││  ││- config       ││  ││- config       ││  ││Driver│││
│ ││- config       ││  ││- filePath     ││  ││- filePath     ││  ││      │││
│ ││               ││  ││- fileHandle   ││  ││- file         ││  ││- cli-│││
│ ││+ Initialize() ││  ││- writer       ││  ││- sheetName    ││  ││  ent │││
│ ││+ LogMetrics() ││  ││- headerWritten││  ││- currentRow   ││  ││- log-│││
│ ││+ LogInfo()    ││  ││               ││  ││               ││  ││  Grp │││
│ ││+ LogError()   ││  ││+ Initialize() ││  ││+ Initialize() ││  ││- seq-│││
│ ││+ Close()      ││  ││+ LogMetrics() ││  ││+ LogMetrics() ││  ││  Tok │││
│ │└───────────────┘│  ││+ LogInfo()    ││  ││+ LogInfo()    ││  ││      │││
│ └─────────────────┘  ││+ LogError()   ││  ││+ LogError()   ││  ││+ Ini-│││
│                      ││+ Close()      ││  ││+ Close()      ││  ││  tia-│││
│                      ││+ RotateFile() ││  ││+ CreateChart()││  ││  lize│││
│                      │└───────────────┘│  │└───────────────┘│  ││+ Log-│││
│                      └─────────────────┘  └─────────────────┘  ││  Met-│││
│                                                               ││  rics│││
│                                                               ││+ Log-│││
│                                                               ││  Info│││
│                                                               ││+ Log-│││
│                                                               ││  Err │││
│                                                               ││+ Clo-│││
│                                                               ││  se  │││
│                                                               │└──────┘││
│                                                               └────────┘│
└─────────────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    EXTERNAL DEPENDENCIES                                  │
│                                                                           │
│ ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌────────┐│
│ │   logrus        │  │   csv           │  │  excelize/v2    │  │ aws-   ││
│ │  (Sirupsen)     │  │ (standard lib)  │  │    (xuri)       │  │ sdk-v2 ││
│ │                 │  │                 │  │                 │  │        ││
│ │ - Logger        │  │ - Writer        │  │ - File          │  │ - CW   ││
│ │ - JSONFormatter │  │ - Reader        │  │ - Style         │  │  Logs  ││
│ │ - Fields        │  │                 │  │ - Chart         │  │ - Config││
│ └─────────────────┘  └─────────────────┘  └─────────────────┘  └────────┘│
└─────────────────────────────────────────────────────────────────────────┘
```

## Interface Implementation Pattern

```
LogDriver Interface
      ↑
      │ implements
      │
┌─────┼─────┬─────────┬─────────┬──────────┐
│     │     │         │         │          │
│     ▼     ▼         ▼         ▼          ▼
│ Console  CSV     Excel   CloudWatch   [Future]
│ Driver  Driver   Driver    Driver     Drivers
│     │     │         │         │          │
│     ▼     ▼         ▼         ▼          ▼
│ stdout  .csv     .xlsx    AWS CW      [Other]
│         files   files     Logs        Systems
```

## Data Flow Architecture

```
Monitor.monitorContainers()
         │
         ▼
ContainerMetrics[] (slice)
         │
         ▼
LogManager.LogMetrics()
         │
         ▼
┌────────┼────────────────────────────────┐
│        ▼                                │
│   For each driver in drivers[]          │
│        │                                │
│        ▼                                │
│   driver.LogMetrics(metrics)            │
│        │                                │
│        ▼                                │
│   ┌─────────┬─────────┬─────────────┐   │
│   ▼         ▼         ▼             ▼   │
│Console   CSV       Excel      CloudWatch│
│stdout   file.csv  workbook.xlsx  AWS    │
└─────────────────────────────────────────┘
```

## Configuration Flow

```
Environment Variables    OR    config.json
         │                           │
         ▼                           ▼
LogManager.LoadConfigFromEnv()  OR  LoadConfigFromFile()
         │                           │
         └─────────┬─────────────────┘
                   ▼
            LogConfig[] array
                   │
                   ▼
            For each config:
             LogManager.AddDriver()
                   │
                   ▼
            Switch config.Driver:
            ┌─────┼──────────────────┐
            ▼     ▼      ▼     ▼     ▼
        Console CSV  Excel CloudWatch NewDriver()
            │     │      │     │     │
            ▼     ▼      ▼     ▼     ▼
        driver.Initialize()
            │     │      │     │     │
            ▼     ▼      ▼     ▼     ▼
        Added to LogManager.drivers[]
```

## Polymorphism in Action

```go
// LogDriver interface enables polymorphism
type LogDriver interface {
    Initialize() error
    LogMetrics([]ContainerMetrics) error
    LogInfo(string, map[string]interface{}) error
    LogError(string, error) error
    Close() error
}

// LogManager treats all drivers uniformly
func (lm *LogManager) LogMetrics(metrics []ContainerMetrics) error {
    for _, driver := range lm.drivers {  // Each driver implements LogDriver
        driver.LogMetrics(metrics)       // Polymorphic call
    }
}
```

## Key Design Patterns

1. **Strategy Pattern**: Different logging drivers (console, CSV, Excel, CloudWatch)
2. **Factory Pattern**: LogManager creates appropriate drivers based on config
3. **Interface Segregation**: LogDriver interface defines common contract
4. **Dependency Injection**: Monitor receives LogManager, not concrete drivers
5. **Configuration Pattern**: External config drives behavior
6. **Error Handling**: Go's explicit error handling throughout
7. **Resource Management**: defer statements ensure proper cleanup
