# 🎪 Event Management Backend System

[![Go Version](https://img.shields.io/badge/Go-1.25.1-blue.svg)](https://golang.org/)
[![Live Demo](https://img.shields.io/badge/Live-Demo-green.svg)](https://evently-api.mitshah.dev/)
[![API Docs](https://img.shields.io/badge/API-Docs-orange.svg)](https://evently-api.mitshah.dev/docs)

> A scalable, high-performance backend system for event management and ticket booking that handles concurrency, prevents overselling, and provides comprehensive analytics.

## 🌟 Overview

It is a robust backend system designed to handle large-scale event bookings with thousands of concurrent users. Built with **Go**, **PostgreSQL**, **Redis**, and **Apache Kafka**, it provides a complete solution for event organizers and attendees.

**🔗 Quick Links:**

- 🌍 **Live API**: [https://evently-api.mitshah.dev/](https://evently-api.mitshah.dev/)
- 📖 **API Documentation**: [https://evently-api.mitshah.dev/docs](https://evently-api.mitshah.dev/docs)
- 🐙 **GitHub Repository**: [https://github.com/Mitshah2406/event-management-backend](https://github.com/Mitshah2406/event-management-backend)
- 📝 **Postman Collection**: [Complete_API_Collection.postman_collection.json](Complete_API_Collection.postman_collection.json)
- 🎥 **Video Demo**: _[Coming Soon]_

---

## ✨ Key Features

### 👤 **User Features**

- **Event Discovery**: Browse upcoming events with detailed information
- **Real-time Booking**: Book tickets with seat-level selection
- **Waitlist Management**: Join waitlists for sold-out events
- **Booking History**: Track all bookings and cancellations
- **Smart Notifications**: Email alerts for waitlist updates

### 👨‍💼 **Admin Features**

- **Event Management**: Create, update, and manage events
- **Venue Configuration**: Design venue layouts with sections and pricing
- **Analytics Dashboard**: Comprehensive booking and revenue analytics
- **Cancellation Policies**: Flexible cancellation rules per event
- **Real-time Monitoring**: Track bookings, waitlists, and system health

### 🏗️ **System Features**

- **Concurrency Control**: Handles thousands of simultaneous bookings
- **Race Condition Prevention**: Optimistic locking and atomic operations
- **Scalable Architecture**: Microservices with Redis caching
- **Real-time Updates**: Kafka-based event streaming
- **Production Ready**: Docker deployment with health monitoring

---

## 🏗️ Architecture Overview

### High-Level System Architecture

![High Level Architecture](<./High%20Level%20Architecture%20(Business%20Flow).png>)

The system follows a **microservices architecture** with clear separation of concerns:

- **API Gateway**: Gin-based REST API with middleware for authentication, rate limiting, and CORS
- **Business Logic**: Modular services for events, bookings, venues, waitlists, and analytics
- **Data Layer**: PostgreSQL for persistent data, Redis for caching and seat holds
- **Message Queue**: Apache Kafka for asynchronous notifications and event processing
- **External Services**: Email service integration for notifications

### Entity Relationship Diagram

![ER Diagram](./ER%20Diagram.png)

**Core Entities:**

- **Users**: Authentication and role-based access control
- **Events**: Event information with venue and capacity management
- **Venues**: Template-based venue designs with sections and seats
- **Bookings**: Booking lifecycle with versioning for concurrent updates
- **Waitlists**: Queue management for sold-out events
- **Cancellation Policies**: Flexible rules for different event types

---

## 🔄 Business Flow Diagrams

### Booking Success Flow

![Booking Success Flow](./Booking%20Success%20Flow.png)
**Overview:**  
This flow handles the complete lifecycle of a successful booking from seat discovery to confirmation, ensuring **data consistency**, **atomic updates**, and **real-time notifications**.

**Step-by-Step Process:**

1. **Seat Discovery**

   - User browses available seats categorized by section, type, and pricing.
   - Real-time availability ensures only free seats are shown.

2. **Seat Hold**

   - Temporary reservation of selected seats with a **10-minute expiry**.
   - Uses **optimistic locking** to prevent double-booking in concurrent scenarios.
   - Provides a buffer for users to confirm before the hold expires.

3. **Booking Confirmation**

   - Performs an **atomic transaction**: updates seat inventory, confirms payment(mock), and records the booking.
   - Ensures **consistency** across multiple services (Seat, Booking, Analytics).
   - Prevents race conditions even under high concurrency.

4. **Waitlist Processing**
   - Automatic notifications sent to waitlisted users if seats become available.
   - Queue position tracked with **optimistic updates** to maintain fairness (FIFO).

### Cancellation Flow

![Cancellation Flow](./Cancellation%20Flow.png)

**Overview:**  
This flow ensures safe handling of cancellations, refunds, and inventory updates, maintaining **atomicity** and notifying affected users.

**Step-by-Step Process:**

1. **Policy Check**

   - Validates cancellation request against event-specific rules.
   - Handles partial refunds, fixed fees, or non-refundable tickets.

2. **Refund Calculation**

   - Computes refund amounts based on timing and policy.
   - Integrates with payment gateways(Mock) for **atomic refund transactions**.

3. **Inventory Release**

   - Frees up cancelled seats using **atomic updates** in the seat inventory.
   - Ensures newly available seats are visible for other users.
   - Uses **optimistic locking** to avoid conflicts with ongoing bookings.

4. **Waitlist Notification**
   - Alerts next eligible users in the waitlist queue.
   - Waitlist promotions also follow **atomic state changes** to maintain FIFO order.

### Waitlist Queue Logic

![Waitlist Queue Logic](./Waitlist%20Queue%20Logic%20Flow.png)

**Overview:**  
The waitlist system manages users efficiently, promoting them when seats become available while maintaining **data integrity** and **fairness**.

**Key Features & Logic:**

- **FIFO Processing**

  - Users notified in the order they joined the queue in batches.

- **Smart Expiry**

  - Automatically removes expired or unresponsive users.
  - Maintains an accurate and clean waitlist state.

- **Batch Processing**

  - Handles multiple cancellations and promotions in bulk.
  - All updates happen via **atomic transactions** to ensure consistency.

- **Real-time Updates**
  - Users receive live status updates on their queue position.
  - Prevents conflicts or double-bookings even during high-demand events.

---

## 🛠️ Tech Stack

| Category           | Technology      | Purpose                                   |
| ------------------ | --------------- | ----------------------------------------- |
| **Language**       | Go 1.25.1       | High-performance backend development      |
| **Framework**      | Gin             | HTTP routing and middleware               |
| **Database**       | PostgreSQL      | Primary data storage with ACID compliance |
| **Cache**          | Redis           | Session management and seat holds         |
| **Message Queue**  | Apache Kafka    | Asynchronous event processing             |
| **ORM**            | GORM            | Database operations and migrations        |
| **Authentication** | JWT             | Stateless authentication                  |
| **Documentation**  | OpenAPI/Swagger | API documentation                         |
| **Deployment**     | Docker          | Containerized deployment                  |
| **Cloud**          | Hostinger VPS   | Production hosting                        |

---

## 📚 API Structure & Documentation

### Base URL

- **Development**: `http://localhost:8080/api/v1`
- **Production**: `https://evently-api.mitshah.dev/api/v1`

### Authentication

All protected endpoints require a JWT token in the Authorization header:

```
Authorization: Bearer <your-jwt-token>
```

### API Endpoints Overview

#### 🔐 Authentication

| Method | Endpoint                | Description          | Access        |
| ------ | ----------------------- | -------------------- | ------------- |
| `POST` | `/auth/register`        | User registration    | Public        |
| `POST` | `/auth/login`           | User login           | Public        |
| `POST` | `/auth/refresh`         | Refresh JWT token    | Authenticated |
| `POST` | `/auth/change-password` | Change user password | Authenticated |

#### 🎪 Events

| Method   | Endpoint                | Description            | Access        |
| -------- | ----------------------- | ---------------------- | ------------- |
| `GET`    | `/events`               | Browse all events      | Public        |
| `GET`    | `/events/{id}`          | Get event details      | Public        |
| `GET`    | `/events/{id}/sections` | Get event venue layout | Authenticated |
| `POST`   | `/admin/events`         | Create new event       | Admin         |
| `PUT`    | `/admin/events/{id}`    | Update event           | Admin         |
| `DELETE` | `/admin/events/{id}`    | Delete event           | Admin         |

#### 🏟️ Venues & Seats

| Method   | Endpoint                               | Description            | Access        |
| -------- | -------------------------------------- | ---------------------- | ------------- |
| `GET`    | `/admin/venue-templates`               | List venue templates   | Admin         |
| `POST`   | `/admin/venue-templates`               | Create venue template  | Admin         |
| `GET`    | `/admin/venue-templates/{id}/sections` | Get template sections  | Admin         |
| `POST`   | `/seats/hold`                          | Hold seats for booking | Authenticated |
| `DELETE` | `/seats/hold/{holdId}`                 | Release seat hold      | Authenticated |
| `GET`    | `/seats/hold/{holdId}/validate`        | Validate seat hold     | Authenticated |

#### 🎫 Bookings

| Method | Endpoint                | Description         | Access        |
| ------ | ----------------------- | ------------------- | ------------- |
| `POST` | `/bookings/confirm`     | Confirm booking     | Authenticated |
| `GET`  | `/bookings/{id}`        | Get booking details | Authenticated |
| `POST` | `/bookings/{id}/cancel` | Cancel booking      | Authenticated |
| `GET`  | `/users/bookings`       | Get user bookings   | Authenticated |

#### ⏰ Waitlist

| Method   | Endpoint                            | Description          | Access        |
| -------- | ----------------------------------- | -------------------- | ------------- |
| `POST`   | `/waitlist`                         | Join event waitlist  | Authenticated |
| `DELETE` | `/waitlist/{eventId}`               | Leave waitlist       | Authenticated |
| `GET`    | `/waitlist/status/{eventId}`        | Get waitlist status  | Authenticated |
| `GET`    | `/admin/waitlist/entries/{eventId}` | Get waitlist entries | Admin         |
| `POST`   | `/admin/waitlist/notify/{eventId}`  | Notify next in line  | Admin         |

#### 📊 Analytics

| Method | Endpoint                          | Description              | Access |
| ------ | --------------------------------- | ------------------------ | ------ |
| `GET`  | `/admin/events/analytics`         | Overall event analytics  | Admin  |
| `GET`  | `/admin/events/{id}/analytics`    | Specific event analytics | Admin  |
| `GET`  | `/admin/analytics/revenue`        | Revenue analytics        | Admin  |
| `GET`  | `/admin/analytics/popular-events` | Popular events report    | Admin  |

#### 🚫 Cancellation Management

| Method | Endpoint                                 | Description                  | Access        |
| ------ | ---------------------------------------- | ---------------------------- | ------------- |
| `POST` | `/admin/events/{id}/cancellation-policy` | Create cancellation policy   | Admin         |
| `GET`  | `/admin/events/{id}/cancellation-policy` | Get cancellation policy      | Admin         |
| `POST` | `/bookings/{id}/request-cancel`          | Request booking cancellation | Authenticated |

### 📋 Sample API Requests

#### User Registration

```bash
curl -X POST https://evently-api.mitshah.dev/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "John",
    "last_name": "Doe",
    "email": "john@example.com",
    "password": "securepass123",
    "role": "USER"
  }'
```

#### Browse Events

```bash
curl -X GET "https://evently-api.mitshah.dev/api/v1/events?limit=10&offset=0"
```

#### Hold Seats

```bash
curl -X POST https://evently-api.mitshah.dev/api/v1/seats/hold \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "seat_ids": ["seat-1", "seat-2"]
  }'
```

#### Confirm Booking

```bash
curl -X POST https://evently-api.mitshah.dev/api/v1/bookings/confirm \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "hold_id": "hold-12345"
  }'
```

---

## 🚀 Getting Started

### Prerequisites

- **Go 1.25.1+**
- **Docker & Docker Compose**
- **Make**

### Development Setup

#### 1️⃣ Clone the Repository

```bash
git clone https://github.com/Mitshah2406/event-management-backend.git
cd evently-backend/backend
```

#### 2️⃣ Environment Configuration

```bash
# Create environment file
cp .env.example .env

# Edit the .env file with your configuration
nano .env
```

**Environment Variables:**

```env
# Server Configuration
PORT=8080
API_VERSION=v1

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_NAME=evently_db
DB_USER=evently_user
DB_PASSWORD=evently_password

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key
JWT_EXPIRY=24h

# Kafka Configuration
KAFKA_BROKER=localhost:9092
KAFKA_TOPIC_WAITLIST=waitlist-notifications

# Email Configuration (Optional)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
```

#### 3️⃣ Quick Start with Makefile

```bash
# Setup everything (dependencies, database, environment)
make setup

# Start development server with live reload
make dev

# Or run without live reload
make run
```

#### 5️⃣ Verify Setup

```bash
# Check health
curl http://localhost:8080/health

# View API documentation
open http://localhost:8080/docs
```

### 🗃️ Database Seeding

The project includes comprehensive seed data for testing:

```bash
# Seed database with sample data (⚠️ This will clean existing data)
make seed
```

**Seed Data Includes:**

- **3 Users**: 1 Admin, 2 Regular users
- **2 Venue Templates**: Theater (26 seats), Conference Hall (28 seats)
- **6 Events**: Various upcoming events with different pricing
- **6 Cancellation Policies**: Different rules and fee structures

**Default Login Credentials:**

```
Admin: admin@gmail.com / qwerty
User1: mitshah2406@gmail.com / qwerty
User2: mitshah2406.work@gmail.com / qwerty
```

---

## 🐳 Production Deployment

### Docker Production Setup

#### 1️⃣ Build Production Images

```bash
# Build all production images
make prod-build
```

#### 2️⃣ Start Production Environment

```bash
# Start all services in production mode
make prod-up
```

#### 3️⃣ Monitor Services

```bash
# Check service status and health
make prod-status

# View application logs
make prod-logs-app

# View all service logs
make prod-logs
```

#### 4️⃣ Production URLs

After starting, services are available at:

- **🌐 Application**: [http://localhost:9000](http://localhost:9000)
- **❤️ Health Check**: [http://localhost:9000/health](http://localhost:9000/health)
- **📖 API Docs**: [http://localhost:9000/docs](http://localhost:9000/docs)
- **🗄️ PostgreSQL**: localhost:9001
- **🔴 Redis**: localhost:9002
- **📨 Kafka**: localhost:9003

### VPS Deployment (Hostinger)

Our production deployment uses **Docker Compose** on a **Hostinger VPS**:

## 🛠️ Development Commands

### Makefile Commands

```bash
# Development
make dev          # Start with live reload
make run          # Start without reload
make build        # Build binary
make test         # Run tests
make clean        # Clean build artifacts

# Database
make seed         # Seed sample data
make migrate      # Run migrations

# Docker Development
make docker-up    # Start dev services
make docker-down  # Stop dev services

# Docker Production
make prod-build   # Build prod images
make prod-up      # Start prod environment
make prod-down    # Stop prod environment
make prod-status  # Check service health
make prod-logs    # View all logs

# Database Connections
make prod-connect-db     # Connect to PostgreSQL
make prod-connect-redis  # Connect to Redis
```

---

## 🔧 Configuration

### Environment Variables Reference

| Variable         | Description       | Default          | Required |
| ---------------- | ----------------- | ---------------- | -------- |
| `PORT`           | Server port       | `8080`           | No       |
| `API_VERSION`    | API version       | `v1`             | No       |
| `DB_HOST`        | PostgreSQL host   | `localhost`      | Yes      |
| `DB_PORT`        | PostgreSQL port   | `5432`           | Yes      |
| `DB_NAME`        | Database name     | `evently_db`     | Yes      |
| `DB_USER`        | Database user     | `evently_user`   | Yes      |
| `DB_PASSWORD`    | Database password | -                | Yes      |
| `REDIS_HOST`     | Redis host        | `localhost`      | Yes      |
| `REDIS_PORT`     | Redis port        | `6379`           | Yes      |
| `REDIS_PASSWORD` | Redis password    | -                | No       |
| `JWT_SECRET`     | JWT signing key   | -                | Yes      |
| `JWT_EXPIRY`     | Token expiry      | `24h`            | No       |
| `KAFKA_BROKER`   | Kafka broker URL  | `localhost:9092` | Yes      |
| `SMTP_HOST`      | Email SMTP host   | -                | No       |
| `SMTP_USERNAME`  | Email username    | -                | No       |
| `SMTP_PASSWORD`  | Email password    | -                | No       |

### Docker Compose Services

#### Development (`docker-compose.dev.yml`)

- **PostgreSQL**: Port 5432
- **Redis**: Port 6379
- **Kafka**: Port 9092
- **Zookeeper**: Port 2181

#### Production (`docker-compose.prod.yml`)

- **Application**: Port 9000
- **PostgreSQL**: Port 9001 (external)
- **Redis**: Port 9002 (external)
- **Kafka**: Port 9003 (external)

---

## 🔍 Monitoring & Debugging

### Health Checks

```bash
# Application health
curl https://evently-api.mitshah.dev/health

# Individual service status
make prod-status
```

### Logging

```bash
# Application logs
make prod-logs-app

# Database logs
make prod-logs-db

# All service logs
make prod-logs
```

### Database Access

```bash
# Production database connection
make prod-connect-db

# Redis connection
make prod-connect-redis

# Connection details
make prod-db-info
make prod-redis-info
```

### Performance Monitoring

- **Response Times**: Built-in Gin middleware logging
- **Database Queries**: GORM query logging
- **Redis Operations**: Connection pool monitoring
- **Memory Usage**: Go runtime metrics
- **Concurrent Requests**: Rate limiting metrics

---

## 🚀 Performance & Scalability

### Concurrency Handling

- **Optimistic Locking**: Version-based conflict resolution
- **Atomic Operations**: Redis-based seat holds with TTL
- **Database Transactions**: ACID compliance for bookings
- **Connection Pooling**: Efficient resource utilization

### Caching Strategy

- **Redis Cache**: Event data, user sessions, seat availability
- **Cache Invalidation**: Smart cache updates on data changes
- **TTL Management**: Automatic cleanup of expired data

### Scalability Features

- **Horizontal Scaling**: Stateless application design
- **Database Sharding**: Partition strategy for large datasets
- **Message Queues**: Asynchronous processing with Kafka
- **CDN Ready**: Static asset optimization
- **Rate Limiting**: Per-user and global rate limits

---

## 🔒 Security

### Authentication & Authorization

- **JWT Tokens**: Stateless authentication
- **Role-Based Access**: USER and ADMIN roles
- **Token Expiry**: Configurable expiration times
- **Refresh Tokens**: Secure token renewal

### Data Protection

- **Password Hashing**: bcrypt with salt
- **SQL Injection**: Parameterized queries with GORM
- **XSS Protection**: Input validation and sanitization
- **CORS**: Configurable cross-origin requests
- **Rate Limiting**: DDoS protection

### Production Security

- **HTTPS/TLS**: SSL certificate with Let's Encrypt
- **Environment Variables**: Secure configuration management
- **Database Connections**: SSL-enabled connections

---

## 📖 Project Structure

```
evently-backend/
├── README.md                           # This file
├── Evently-Task.txt                    # Original task requirements
├── SEED_SUMMARY.md                     # Database seed information
├── *.png                              # Architecture diagrams
├── backend/                           # Main application directory
│   ├── go.mod                         # Go module definition
│   ├── Makefile                       # Development commands
│   ├── .env.example                   # Environment template
│   │
│   ├── server/                        # Application entry point
│   │   └── main.go                    # Server initialization
│   │
│   ├── api/                           # API layer
│   │   └── routes/
│   │       └── router.go              # Main router and setup
│   │
│   ├── internal/                      # Business logic modules
│   │   ├── auth/                      # Authentication service
│   │   │   ├── controller.go          # Auth endpoints
│   │   │   ├── service.go             # Auth business logic
│   │   │   ├── repository.go          # Auth data access
│   │   │   └── models.go              # Auth data models
│   │   │
│   │   ├── events/                    # Event management
│   │   ├── bookings/                  # Booking system
│   │   ├── seats/                     # Seat management
│   │   ├── venues/                    # Venue templates
│   │   ├── waitlist/                  # Waitlist management
│   │   ├── cancellation/              # Cancellation handling
│   │   ├── analytics/                 # Analytics service
│   │   ├── notifications/             # Email notifications
│   │   └── shared/                    # Shared utilities
│   │       ├── config/                # Configuration management
│   │       ├── database/              # Database connections
│   │       ├── middleware/            # HTTP middleware
│   │       └── utils/                 # Common utilities
│   │
│   ├── pkg/                           # External packages
│   │   ├── cache/                     # Redis service
│   │   ├── logger/                    # Logging utilities
│   │   └── ratelimit/                 # Rate limiting
│   │
│   ├── cmd/                           # CLI commands
│   │   ├── seed/                      # Database seeding
│   │   │   └── main.go
│   │   └── emailtest/                 # Email testing
│   │       └── main.go
│   │
│   ├── deployments/                   # Deployment configuration
│   │   └── docker/
│   │       ├── docker-compose.dev.yml # Development environment
│   │       ├── docker-compose.prod.yml# Production environment
│   │       ├── Dockerfile.dev         # Development image
│   │       ├── Dockerfile.prod        # Production image
│   │       └── init.sql               # Database initialization
│   │
│   ├── docs/                          # API documentation
│   │   └── swagger.yaml               # OpenAPI specification
│   │
│   └── tmp/                           # Temporary files (gitignored)
│       └── main                       # Development binary
```

---

## 👨‍💻 Author & Contact

**Mit Shah**

- 🐙 **GitHub**: [@Mitshah2406](https://github.com/Mitshah2406)
- 📧 **Email**: mitshah2406.work@gmail.com
- 🌐 **Portfolio**: [mitshah.dev](https://mitshah.dev)

---

## 🎯 Future Enhancements

### Planned Features

- [ ] **Multi-tenant Architecture**: Support multiple event organizers
- [ ] **Payment Gateway**: Stripe/Razorpay integration
- [ ] **Real-time Updates**: WebSocket for live seat availability
- [ ] **Social Features**: User reviews and event sharing
- [ ] **Loyalty Program**: Points and rewards system

### Technical Improvements

- [ ] **Microservices**: Break into smaller, independent services
- [ ] **Kubernetes**: Container orchestration for scaling
- [ ] **Event Sourcing**: Complete audit trail for all operations
- [ ] **CQRS Pattern**: Separate read/write models for better performance

---

### Technology Choices Rationale

| Choice         | Reason                                                         |
| -------------- | -------------------------------------------------------------- |
| **Go**         | High concurrency, fast compilation, excellent standard library |
| **PostgreSQL** | ACID compliance, complex queries, JSON support                 |
| **Redis**      | Sub-millisecond response times, atomic operations              |
| **Kafka**      | Reliable message delivery, horizontal scaling                  |
| **Docker**     | Consistent deployments, easy scaling, development parity       |
| **JWT**        | Stateless authentication, mobile-friendly, scalable            |

---

## 🔗 Additional Resources

### Documentation

- [Go Documentation](https://golang.org/doc/)
- [Gin Framework Guide](https://gin-gonic.com/docs/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Redis Documentation](https://redis.io/documentation)
- [Docker Documentation](https://docs.docker.com/)

### API Testing

- **Postman Collection**: [Complete_API_Collection.postman_collection.json](Complete_API_Collection.postman_collection.json)
- **OpenAPI Spec**: [swagger.yaml](./backend/docs/swagger.yaml)

---

_Last Updated: September 2025 | Version: 1.0.0_
