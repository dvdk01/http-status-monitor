# HTTP Status Monitor

A tool for monitoring HTTP statuses of specified URLs.

## Project Structure

```
.
├── cmd/
│   └── http-status-monitor/    # Main application
├── internal/                   # Internal packages
│   ├── application/           # Application logic
│   ├── monitor/              # Monitoring logic
│   ├── processor/            # Data processing
│   ├── schema/               # Data structures
│   └── validator/            # Input validation
├── e2e/                      # End-to-end tests
├── docs/                     # Documentation
└── Dockerfile               # Container configuration
```

## Installation

### Using Go

```bash
go install github.com/dvdk01/http-status-monitor/cmd/http-status-monitor@latest
```

### Using Docker

```bash
# Build the image
docker build -t http-status-monitor .
```

## Usage

### Using Go Binary

```bash
http-status-monitor <url1> <url2> ... <urlN>
```

### Using Docker

```bash
# Run with URLs as arguments
docker run --rm http-status-monitor <url1> <url2> ... <urlN>

# Example Run
docker run --rm http-status-monitor https://github.com https://root.cz https://claude.ai/chat/4b6f968f-2fc1-4228-a19a-6c330179dd6a https://mdecoder.com/decode/wbagl63423dp66538 https://www.thingiverse.com/thing:3973692w12128 https://httpstat.us/500 https://gfjdhgdfjhgjkdf.com/fdfd https://www.bazos.cz/rss.php\?rub\=au https://httpstat.us/100 https://httpstat.us/101
```
