# Evently Backend API Documentation

This project includes comprehensive Swagger/OpenAPI documentation for all API endpoints.

## 🔗 Accessing the Documentation

Once the server is running, you can access the API documentation at:

- **Swagger UI**: http://localhost:8080/swagger/index.html
- **Raw YAML**: http://localhost:8080/docs/swagger.yaml

## 📚 API Overview

The Evently Backend API provides the following main features:

### 🔐 Authentication

- User registration and login
- JWT token-based authentication
- Role-based access control (USER/ADMIN)
- Password management

### 🎪 Event Management

- Browse public events
- Admin event creation/management
- Event analytics and reporting
- Tag-based categorization

### 🏟️ Venue & Seating

- Venue template management
- Dynamic seat generation
- Real-time seat availability
- Temporary seat holding mechanism

### 🎫 Booking System

- Secure booking confirmation
- Booking history and management
- Cancellation handling
- Payment integration support

### ⏰ Waitlist Management

- Join waitlists for sold-out events
- Automatic notifications when seats become available
- Priority-based seat allocation

### 📊 Analytics

- Event performance metrics
- Revenue tracking
- Conversion rate analytics
- Admin dashboard insights

## 🚀 Quick Start

1. **Start the server:**

   ```bash
   make run
   # or
   go run server/main.go
   ```

2. **Access Swagger UI:**
   Open http://localhost:8080/swagger/index.html in your browser

3. **Test endpoints:**
   - Use the "Try it out" feature in Swagger UI
   - Start with `/health` to verify the API is running
   - Register a user via `/auth/register`
   - Login to get an access token via `/auth/login`

## 🔑 Authentication

Most endpoints require authentication. To use protected endpoints:

1. **Register or Login** to get an access token
2. **Click "Authorize"** in Swagger UI
3. **Enter**: `Bearer <your-access-token>`
4. **Click "Authorize"** to apply the token

## 📝 API Structure

### Public Endpoints

- Health checks
- Event browsing
- Tag listing

### User Endpoints (Requires Authentication)

- Seat booking and management
- Personal booking history
- Waitlist operations
- Profile management

### Admin Endpoints (Requires Admin Role)

- Event management
- Venue template management
- Analytics access
- Tag management

## 🏷️ Response Format

All API responses follow a consistent format:

```json
{
  "status": "success|error",
  "message": "Human readable message",
  "data": {}, // Response data (null on error)
  "error": "Error details (only on error)"
}
```

## 🔧 Error Handling

The API uses standard HTTP status codes:

- `200` - Success
- `201` - Created
- `400` - Bad Request (validation errors)
- `401` - Unauthorized (missing/invalid token)
- `403` - Forbidden (insufficient permissions)
- `404` - Not Found
- `409` - Conflict (duplicate resources)
- `500` - Internal Server Error

## 📋 Rate Limiting

The API implements rate limiting on various endpoint categories:

- Public endpoints: Higher limits for browsing
- Auth endpoints: Moderate limits for security
- Booking endpoints: Lower limits for fairness
- Admin endpoints: Higher limits for operations

Rate limit headers are included in responses:

- `X-RateLimit-Limit`
- `X-RateLimit-Remaining`
- `X-RateLimit-Reset`

## 🎯 Key Workflows

### 1. Event Booking Flow

1. Browse events (`GET /events`)
2. Get event sections (`GET /events/{id}/sections`)
3. Check seat availability (`GET /seats/available`)
4. Hold seats (`POST /seats/hold`)
5. Confirm booking (`POST /bookings/confirm`)

### 2. Waitlist Flow (when event is sold out)

1. Join waitlist (`POST /waitlist/join`)
2. Receive notification when seats available
3. Book seats within notification window
4. Mark waitlist entry as converted

### 3. Admin Event Management

1. Create venue template (`POST /admin/venue-templates`)
2. Add sections to template (`POST /admin/venue-templates/{id}/sections`)
3. Create event (`POST /admin/events`)
4. Monitor analytics (`GET /admin/events/{id}/analytics`)

## 📞 Support

For API support or questions:

- **Email**: api@evently.com
- **Documentation**: http://localhost:8080/swagger/index.html
- **GitHub**: [Project Repository](https://github.com/evently-platform/backend)

---

## 🔄 Development Notes

### Adding New Endpoints

When adding new endpoints to the API:

1. **Update the YAML file**: Add your endpoint definition to `/docs/swagger.yaml`
2. **Follow the existing pattern**: Use consistent schemas and response formats
3. **Test in Swagger UI**: Ensure your endpoint displays correctly
4. **Update this README**: Add any new workflows or important information

### YAML vs Code Annotations

This project uses a standalone YAML file instead of code annotations to maintain:

- ✅ **Clean code** - No documentation clutter in source files
- ✅ **Better maintainability** - Centralized documentation
- ✅ **Team collaboration** - Non-developers can contribute to docs
- ✅ **Version control** - Easy to track documentation changes
- ✅ **Flexibility** - Rich OpenAPI features without code constraints

The YAML file is served directly by the application, providing full OpenAPI 3.0 compliance and rich documentation features.
