#!/usr/bin/env python3
"""
Docker Monitor Architecture Diagram Generator
Uses the diagrams (mingrammer) library to create architecture diagrams as code.
"""

from diagrams import Diagram, Cluster, Node, Edge
from diagrams.programming.language import Go
from diagrams.onprem.container import Docker
from diagrams.generic.storage import Storage
from diagrams.aws.management import Cloudwatch
from diagrams.onprem.compute import Server
from diagrams.onprem.client import Users
import os

# Change to the architecture directory
os.chdir(os.path.dirname(__file__))

with Diagram("Docker Monitor Architecture",
             filename="docker_monitor_architecture",
             show=False,
             direction="TB",
             graph_attr={
                 "fontsize": "20",
                 "bgcolor": "white",
                 "pad": "1.0",
                 "splines": "ortho"
             }):

    # User/Admin interaction
    admin = Users("Administrator")

    # Environment & Configuration
    with Cluster("Configuration Sources"):
        env_vars = Storage("Environment\nVariables")
        yaml_config = Storage("config.yaml\n(YAML Config)")

    # Main Application Container
    with Cluster("Docker Monitor Application"):
        with Cluster("Main Process (main.go)"):
            monitor = Go("Monitor\nStruct")
            constructor = Go("NewMonitor()\nConstructor")
            stats_calc = Go("CPU & Memory\nCalculations")

        with Cluster("Logger Package"):
            log_manager = Go("LogManager\n(Orchestrator)")
            log_interface = Go("LogDriver\nInterface")

        with Cluster("Logging Drivers"):
            console_driver = Go("Console\nDriver")
            csv_driver = Go("CSV\nDriver")
            excel_driver = Go("Excel\nDriver")
            cloudwatch_driver = Go("CloudWatch\nDriver")

    # External Systems
    with Cluster("Docker Environment"):
        docker_daemon = Docker("Docker\nDaemon")
        docker_api = Docker("Docker\nAPI")
        containers = [
            Docker("App\nContainer"),
            Docker("Redis\nContainer"),
            Docker("DB\nContainer")
        ]

    # Output Destinations
    with Cluster("Output Destinations"):
        console_out = Server("Console\n(stdout/stderr)")
        csv_files = Storage("CSV Files\n(/logs/*.csv)")
        excel_files = Storage("Excel Files\n(/logs/*.xlsx)")
        aws_cloudwatch = Cloudwatch("AWS CloudWatch\nLogs")

    # Configuration flow
    admin >> env_vars
    admin >> yaml_config

    # Application initialization
    env_vars >> constructor
    yaml_config >> constructor
    constructor >> monitor
    constructor >> log_manager

    # Docker monitoring flow
    monitor >> Edge(label="Docker Client") >> docker_api
    docker_api >> docker_daemon
    docker_daemon >> containers

    # Statistics collection
    docker_api >> Edge(label="Container Stats") >> monitor
    monitor >> stats_calc

    # Logging flow - Manager orchestrates all drivers
    monitor >> Edge(label="ContainerMetrics") >> log_manager
    log_manager >> log_interface

    # Driver implementations
    log_interface >> console_driver
    log_interface >> csv_driver
    log_interface >> excel_driver
    log_interface >> cloudwatch_driver

    # Output destinations
    console_driver >> console_out
    csv_driver >> csv_files
    excel_driver >> excel_files
    cloudwatch_driver >> aws_cloudwatch

print("✅ Architecture diagram generated: docker_monitor_architecture.png")