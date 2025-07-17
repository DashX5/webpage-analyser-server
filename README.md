# Webpage Analyzer Server

A production-ready Go microservice that analyzes web pages and provides detailed insights about their structure, content, and accessibility features. The service offers concurrent link checking, intelligent caching, comprehensive metrics, and rate limiting capabilities.

## 🚀 Project Overview

The Webpage Analyzer Server is a high-performance REST API service designed to analyze web pages and extract meaningful information including:

- **HTML version detection** - Automatically detects HTML document version
- **Content analysis** - Extracts page titles and heading structures
- **Link validation** - Concurrent checking of internal/external links with accessibility status
- **Security analysis** - Intelligent login form detection using scoring algorithms
- **Performance optimization** - Redis caching with configurable TTL
- **Monitoring & Metrics** - Prometheus metrics integration
- **Rate limiting** - Configurable per-IP rate limiting protection

## 🛠️ Technology Stack

### Backend Technologies
- **Language**: [Go 1.24](https://golang.org/)
- **Web Framework**: [Gin](https://github.com/gin-gonic/gin) - HTTP web framework
- **HTML Parser**: [goquery](https://github.com/PuerkitoBio/goquery) - jQuery-like HTML document traversal
- **Caching**: [Redis](https://redis.io/) with [go-redis](https://github.com/redis/go-redis) client
- **Configuration**: [Viper](https://github.com/spf13/viper) - Configuration management
- **Logging**: [Zap](https://github.com/uber-go/zap) - Structured logging
- **Metrics**: [Prometheus](https://prometheus.io/) - Monitoring and alerting
- **Testing**: [Testify](https://github.com/stretchr/testify) - Testing toolkit
- **Validation**: [Validator](https://github.com/go-playground/validator) - Request validation

### Frontend Technologies
- **Template Engine**: HTML templates with Go
- **Styling**: [Tailwind CSS](https://tailwindcss.com/) - Utility-first CSS framework
- **JavaScript**: Vanilla JavaScript for API interactions
- **UI Components**: Responsive design with modern UX patterns

### DevOps & Infrastructure
- **Containerization**: [Docker](https://www.docker.com/) & [Docker Compose](https://docs.docker.com/compose/)
- **Database**: [Redis 7-alpine](https://hub.docker.com/_/redis) - In-memory data store
- **Development Tools**: [Redis Commander](https://joeferner.github.io/redis-commander/) - Redis GUI (dev environment)
- **Configuration**: Environment-based configuration (dev/prod)
- **Monitoring**: Built-in Prometheus metrics endpoint

## 📋 Prerequisites

Before running the project, ensure you have the following installed:

### Required
- **Docker** (version 20.10+): [Download Docker](https://docs.docker.com/get-docker/)
- **Docker Compose** (version 2.0+): [Docker Compose Installation](https://docs.docker.com/compose/install/)

### Optional (for local development)
- **Go** (version 1.24+): [Download Go](https://golang.org/dl/)
- **Redis** (version 7+): [Redis Installation](https://redis.io/download)
- **Git**: [Download Git](https://git-scm.com/downloads)

## 🔧 External Dependencies

### Core Dependencies
The project uses the following external libraries and services:

#### HTTP & Web Framework
- `github.com/gin-gonic/gin` - Web framework for building APIs
- `github.com/PuerkitoBio/goquery` - HTML document parsing and manipulation

#### Caching & Database
- `github.com/redis/go-redis/v9` - Redis client for Go
- Redis server (containerized) - In-memory data store

#### Configuration & Validation
- `github.com/spf13/viper` - Configuration management
- `github.com/go-playground/validator/v10` - Request validation

#### Logging & Monitoring
- `go.uber.org/zap` - Structured logging
- `github.com/prometheus/client_golang` - Prometheus metrics

#### Utilities
- `golang.org/x/time` - Rate limiting utilities
- `golang.org/x/net` - Network utilities

All dependencies are automatically managed through Go modules and Docker containerization.

## 🚀 Setup Instructions

### Quick Start with Docker (Recommended)

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd webpage-analyser-server
   ```

2. **Choose your environment and start services**

   **For Development:**
   ```bash
   # Start development environment with hot reload and Redis Commander
   docker compose -f docker/compose/docker-compose.yml -f docker/compose/docker-compose.dev.yml up --build
   ```

   **For Production:**
   ```bash
   # Start production environment with optimized settings
   docker compose -f docker/compose/docker-compose.yml -f docker/compose/docker-compose.prod.yml up --build
   ```

3. **Access the application**
   - **API Endpoint**: http://localhost:8080
   - **Web Interface**: http://localhost:8080 (HTML form interface)
   - **Redis Commander** (dev only): http://localhost:8081
   - **Metrics Endpoint**: http://localhost:8080/metrics

### Environment Configuration

Create environment files for different environments:

```bash
# Development environment
cp config-files/dev.yaml config-files/dev.yaml.local
# Edit config-files/dev.yaml.local with your settings

# Production environment  
cp config-files/dev.yaml config-files/prod.yaml
# Edit config-files/prod.yaml with production settings
```

### Docker Services

The application consists of the following services:

| Service | Description | Port | Environment |
|---------|-------------|------|-------------|
| `app` | Main application server | 8080 | All |
| `redis` | Redis cache database | 6379 | All |
| `redis-commander` | Redis GUI interface | 8081 | Development only |

### Configuration Options

Key configuration parameters (see `config-files/dev.yaml`):

```yaml
server:
  port: 8080                    # Server port
  timeout: 30s                  # Request timeout
  mode: debug                   # Server mode (debug/release)

analyzer:
  max_links: 100               # Maximum links to analyze
  link_timeout: 10s            # Timeout for link checking
  max_workers: 20              # Concurrent workers
  max_redirects: 0             # Redirect following

cache:
  enabled: true                # Enable Redis caching
  ttl: 1h                     # Cache time-to-live
  
rate_limit:
  enabled: true                # Enable rate limiting
  requests_per_minute: 60      # Rate limit threshold
```

### Local Development (Optional)

If you prefer to run without Docker:

1. **Install dependencies**
   ```bash
   go mod download
   ```

2. **Start Redis**
   ```bash
   redis-server
   ```

3. **Run the application**
   ```bash
   go run cmd/server/main.go
   ```

## 📖 API Documentation

### Base URL
- **Development**: `http://localhost:8080`
- **Production**: `http://your-domain.com`

### Endpoints

#### 1. Analyze Webpage
Analyzes a webpage and returns detailed information about its structure and content.

**Endpoint**: `POST /api/v1/analyze`

**Headers**:
```
Content-Type: application/json
```

**Request Body**:
```json
{
    "url": "https://example.com"
}
```

**Response** (200 OK):
```json
{
    "url": "https://example.com",
    "html_version": "HTML5",
    "title": "Example Domain",
    "headings": {
        "h1": 1,
        "h2": 2,
        "h3": 0,
        "h4": 0,
        "h5": 0,
        "h6": 0
    },
    "links": {
        "internal": 2,
        "external": 1,
        "inaccessible": 0
    },
    "has_login_form": false,
    "analyzed_at": "2024-03-19T10:30:00Z"
}
```

**Error Responses**:
- `400 Bad Request`: Invalid request format or validation failure
- `500 Internal Server Error`: Server processing error

#### 2. Health Check
Simple health check endpoint.

**Endpoint**: `GET /health`

**Response** (200 OK):
```json
{
    "status": "healthy",
    "timestamp": "2024-03-19T10:30:00Z"
}
```

#### 3. Metrics
Prometheus metrics endpoint for monitoring.

**Endpoint**: `GET /metrics`

**Response**: Prometheus formatted metrics

#### 4. Web Interface
Interactive HTML interface for testing the API.

**Endpoint**: `GET /`

**Response**: HTML page with form interface



## 🔧 Development Tools

### Useful Docker Commands

```bash
# View logs
docker compose -f docker/compose/docker-compose.yml logs -f

# List containers
docker compose -f docker/compose/docker-compose.yml ps

# Stop services
docker compose -f docker/compose/docker-compose.yml down

# Clean up (remove containers, volumes, and images)
docker compose -f docker/compose/docker-compose.yml down -v --rmi all
docker system prune -f

# Rebuild specific service
docker compose -f docker/compose/docker-compose.yml build app
```

### Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...



```



## 📊 Monitoring & Metrics

The application exposes Prometheus metrics at `/metrics` endpoint:

### Available Metrics
- **Request Duration**: HTTP request processing time
- **Cache Hit/Miss Ratio**: Cache performance statistics
- **Link Check Duration**: Time spent checking external links
- **Active Connections**: Current active connections
- **Error Rates**: Application error statistics

### Monitoring Setup
For production monitoring, consider integrating with:
- **Prometheus** - Metrics collection
- **Grafana** - Metrics visualization
- **AlertManager** - Alert notifications

## 🚦 Rate Limiting

The API implements rate limiting to prevent abuse:
- **Default**: 60 requests per minute per IP
- **Configurable**: Adjust via configuration files
- **Headers**: Rate limit status included in response headers

## 🔒 Security Features

- **Input Validation**: Comprehensive request validation
- **Rate Limiting**: Protection against API abuse
- **CORS Configuration**: Configurable cross-origin resource sharing
- **Secure Headers**: Security headers in responses
- **URL Validation**: Prevents access to internal/malicious URLs

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.


---

**Built with ❤️ using Go, Docker, and modern web technologies**