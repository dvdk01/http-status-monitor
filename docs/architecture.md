# HTTP Status Monitor Architecture

## Overview

HTTP Status Monitor is a tool for monitoring website availability and performance. The application is written in Go and uses modern approaches to monitoring and data processing.

## Components

### Monitor
- Performs periodic checks of specified URLs
- Measures response time and response size
- Tracks HTTP status codes

### Processor
- Coordinates work between monitor and display
- Processes data from the monitor
- Ensures proper application shutdown

### Application
- Provides user interface
- Displays real-time statistics
- Formats output for better readability

### Validator
- Verifies the correctness of entered URLs
- Ensures valid input data

## Data Flow

1. User enters a list of URLs
2. Validator verifies the URLs
3. Monitor starts periodic checks of the specified addresses
4. Processor processes data from the monitor
5. Application displays statistics to the user
