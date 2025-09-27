#!/usr/bin/env python3
"""
Docker Monitor Architecture Diagram Generator (Simple Version)
Generates DOT file that can be rendered with Graphviz
"""

import os

# Change to the architecture directory
os.chdir(os.path.dirname(__file__))

dot_content = """
digraph "Docker Monitor Architecture" {
    graph [fontsize=20, bgcolor=white, pad=1.0, splines=ortho, rankdir=TB];
    node [shape=box, style=filled, fontname="Arial"];
    edge [fontname="Arial"];

    // Styling
    subgraph cluster_config {
        label="Configuration Sources";
        style=filled;
        fillcolor=lightblue;
        env_vars [label="Environment\nVariables", fillcolor=lightgray];
        yaml_config [label="config.yaml\n(YAML Config)", fillcolor=lightgray];
    }

    subgraph cluster_main_app {
        label="Docker Monitor Application";
        style=filled;
        fillcolor=lightgreen;

        subgraph cluster_main_process {
            label="Main Process (main.go)";
            style=filled;
            fillcolor=white;
            monitor [label="Monitor\nStruct", fillcolor=yellow];
            constructor [label="NewMonitor()\nConstructor", fillcolor=yellow];
            stats_calc [label="CPU & Memory\nCalculations", fillcolor=yellow];
        }

        subgraph cluster_logger_package {
            label="Logger Package";
            style=filled;
            fillcolor=white;
            log_manager [label="LogManager\n(Orchestrator)", fillcolor=orange];
            log_interface [label="LogDriver\nInterface", fillcolor=orange];
        }

        subgraph cluster_drivers {
            label="Logging Drivers";
            style=filled;
            fillcolor=white;
            console_driver [label="Console\nDriver", fillcolor=pink];
            csv_driver [label="CSV\nDriver", fillcolor=pink];
            excel_driver [label="Excel\nDriver", fillcolor=pink];
            cloudwatch_driver [label="CloudWatch\nDriver", fillcolor=pink];
        }
    }

    subgraph cluster_docker {
        label="Docker Environment";
        style=filled;
        fillcolor=lightcyan;
        docker_daemon [label="Docker\nDaemon", fillcolor=cyan];
        docker_api [label="Docker\nAPI", fillcolor=cyan];
        app_container [label="App\nContainer", fillcolor=cyan];
        redis_container [label="Redis\nContainer", fillcolor=cyan];
        db_container [label="DB\nContainer", fillcolor=cyan];
    }

    subgraph cluster_outputs {
        label="Output Destinations";
        style=filled;
        fillcolor=lightyellow;
        console_out [label="Console\n(stdout/stderr)", fillcolor=wheat];
        csv_files [label="CSV Files\n(/logs/*.csv)", fillcolor=wheat];
        excel_files [label="Excel Files\n(/logs/*.xlsx)", fillcolor=wheat];
        aws_cloudwatch [label="AWS CloudWatch\nLogs", fillcolor=wheat];
    }

    // Admin user
    admin [label="Administrator", fillcolor=lightcoral];

    // Configuration flow
    admin -> env_vars [label="configures"];
    admin -> yaml_config [label="configures"];

    // Application initialization
    env_vars -> constructor [label="environment"];
    yaml_config -> constructor [label="config"];
    constructor -> monitor [label="creates"];
    constructor -> log_manager [label="initializes"];

    // Docker monitoring flow
    monitor -> docker_api [label="Docker Client"];
    docker_api -> docker_daemon [label="API calls"];
    docker_daemon -> app_container [label="manages"];
    docker_daemon -> redis_container [label="manages"];
    docker_daemon -> db_container [label="manages"];

    // Statistics collection
    docker_api -> monitor [label="Container Stats"];
    monitor -> stats_calc [label="processes"];

    // Logging flow
    monitor -> log_manager [label="ContainerMetrics"];
    log_manager -> log_interface [label="orchestrates"];

    // Driver implementations
    log_interface -> console_driver [label="implements"];
    log_interface -> csv_driver [label="implements"];
    log_interface -> excel_driver [label="implements"];
    log_interface -> cloudwatch_driver [label="implements"];

    // Output destinations
    console_driver -> console_out [label="outputs"];
    csv_driver -> csv_files [label="writes"];
    excel_driver -> excel_files [label="writes"];
    cloudwatch_driver -> aws_cloudwatch [label="sends"];
}
"""

# Write DOT file
with open("docker_monitor_architecture.dot", "w") as f:
    f.write(dot_content)

print("DOT file generated: docker_monitor_architecture.dot")
print("To generate PNG image, run:")
print("   dot -Tpng docker_monitor_architecture.dot -o docker_monitor_architecture.png")
print("   (requires Graphviz installation)")