# TAPIR - Test API Rapid

TAPI is a high-performance API testing tool written in Go, designed to handle large numbers of concurrent requests. It supports various HTTP methods and provides real-time statistics of the test execution.

## Features

- Support for multiple HTTP methods (GET, POST, PUT, etc.)
- Concurrent request execution with configurable workers
- Batch processing to handle large numbers of requests
- Real-time statistics display
- Response logging for error status codes
- Progress tracking with success rate calculation

## Installation

1. Make sure you have Go installed on your system
2. Build the executable:
   ```bash
   go build -o tapi.exe main.go
   ```

## Usage

```bash
tapi.exe [flags]
```

### Available Flags

- `-m` HTTP method (default "GET")
- `-w` Number of concurrent workers (default 100)
- `-n` Total number of requests (default 100000)
- `-batch` Batch size for processing requests (default 1000)
- `-url` Target API URL (required)
- `-b` Path to JSON body file (optional)

### Example Commands

- Basic POST request with 100 workers:
  ```bash
  tapi.exe -m post -w 100 -url http://api.example.com/endpoint -b body.json
  ```

- GET request with 1000 total requests:
  ```bash
  tapi.exe -m get -n 1000 -url http://api.example.com/endpoint
  ```

- Custom batch size:
  ```bash
  tapi.exe -m post -w 100 -n 100000 -batch 500 -url http://api.example.com/endpoint -b body.json
  ```

## Output Format

The tool displays real-time statistics in the following format:

```
Processed: X/Y (Z% success) | Status AAA: B/Y | Status BBB: C/Y | Time: Ds
```

Where:
- X: Number of processed requests
- Y: Total number of requests
- Z: Success rate percentage
- AAA: HTTP status code
- B, C: Count of responses for each status code
- D: Elapsed time in seconds

## Error Logging

Error responses (non-200 status codes) are automatically saved in the logs directory:
```
logs/ErrorXXX_timestamp.txt
```
where XXX is the status code and timestamp is the Unix millisecond timestamp.

## System Requirements

- Go 1.15 or higher
- Sufficient system resources for concurrent connections
- Adequate network bandwidth

## Notes

- Adjust the worker count (`-w`) based on your system capabilities
- Monitor system resources during large-scale tests
- Consider network limitations when setting concurrent workers
- Use appropriate batch sizes for optimal performance

## Contributing

Please report any issues or submit contributions through the project repository.