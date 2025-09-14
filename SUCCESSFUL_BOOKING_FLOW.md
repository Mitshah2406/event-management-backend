# Success Booking API Flow - Evently Backend

This document outlines the complete API flow for a successful ticket booking process in the Evently platform. The flow demonstrates the step-by-step process from user authentication to booking confirmation.

## Overview

The booking flow consists of 8 main steps:

1. User Authentication
2. Browse Upcoming Events
3. Get Event Details
4. Check Available Seats
5. Hold Seats (Reserve Temporarily)
6. Verify Hold Status
7. Confirm Booking & Payment
8. View Booking History

## API Flow Steps

### 1. User Authentication

**Endpoint:** `POST /api/v1/users/login`

**Purpose:** Authenticate user and obtain JWT tokens for subsequent requests.

**Request:**

```json
{
  "email": "mitshah2406@gmail.com",
  "password": "qwerty"
}
```

**Response:**

```json
{
  "status": "success",
  "status_code": 200,
  "message": "Login successful",
  "data": {
    "user": {
      "id": "c8d74182-7bb8-48c2-b984-b68f626bf019",
      "first_name": "Mit",
      "last_name": "Shah",
      "email": "mitshah2406@gmail.com",
      "role": "USER",
      "created_at": "2025-09-14T01:41:33.842046+05:30",
      "updated_at": "2025-09-14T01:41:33.842046+05:30"
    },
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 86400
  }
}
```

**Notes:**

- Save the `access_token` for authentication in subsequent requests
- Token expires in 86400 seconds (24 hours)

---

### 2. Browse Upcoming Events

**Endpoint:** `GET /api/v1/events/upcoming`

**Purpose:** Retrieve a list of all upcoming events available for booking.

**Authentication:** None required (public endpoint)

**Response:**

```json
{
  "status": "success",
  "status_code": 200,
  "message": "Upcoming events retrieved successfully",
  "data": [
    {
      "id": "0d359479-818e-424e-87fc-632e6b3db58b",
      "name": "Startup Pitch Night",
      "description": "Watch promising startups pitch their ideas to investors and industry experts.",
      "venue": "Innovation Center",
      "venue_template_id": "cfa0f2be-30b6-4337-ad23-76e7b711d6c1",
      "date_time": "2025-09-29T01:41:33.912313+05:30",
      "total_capacity": 0,
      "booked_count": 0,
      "available_tickets": 0,
      "base_price": 500,
      "status": "published",
      "image_url": "",
      "tags": [
        {
          "id": "ec1681b9-03c8-4ca6-9396-3fc6e49ef962",
          "name": "Technology",
          "slug": "technology",
          "color": "#3B82F6"
        }
      ],
      "created_at": "2025-09-14T01:41:33.912313+05:30",
      "updated_at": "2025-09-14T01:41:33.912313+05:30"
    }
  ]
}
```

---

### 3. Get Event Details

**Endpoint:** `GET /api/v1/events/:eventId`

**Purpose:** Retrieve detailed information about a specific event, including venue sections and seating layout.

**Authentication:** None required

**Example:** `GET /api/v1/events/0d359479-818e-424e-87fc-632e6b3db58b`

**Response:**

```json
{
    "status": "success",
    "status_code": 200,
    "message": "Event retrieved successfully",
    "data": {
        "id": "0d359479-818e-424e-87fc-632e6b3db58b",
        "name": "Startup Pitch Night",
        "description": "Watch promising startups pitch their ideas to investors and industry experts.",
        "venue": "Innovation Center",
        "venue_template_id": "cfa0f2be-30b6-4337-ad23-76e7b711d6c1",
        "venue_sections": [
            {
                "id": "c4ccd0fb-c85f-467b-8fc1-a9fc12852ecb",
                "name": "General",
                "description": "General seating for all attendees",
                "row_start": "B",
                "row_end": "C",
                "seats_per_row": 10,
                "total_seats": 20,
                "price_multiplier": 1,
                "price": 500
            },
            {
                "id": "ffd21768-9ee9-4b03-99b8-9e7c51ec76cd",
                "name": "VIP",
                "description": "VIP seating with premium amenities",
                "row_start": "A",
                "row_end": "A",
                "seats_per_row": 8,
                "total_seats": 8,
                "price_multiplier": 1.5,
                "price": 750
            }
        ],
        "date_time": "2025-09-29T01:41:33.912313+05:30",
        "total_capacity": 0,
        "booked_count": 0,
        "available_tickets": 0,
        "base_price": 500,
        "status": "published",
        "tags": [...],
        "created_at": "2025-09-14T01:41:33.912313+05:30",
        "updated_at": "2025-09-14T01:41:33.912313+05:30"
    }
}
```

---

### 4. Get Available Seats by Section

**Endpoint:** `GET /api/v1/sections/:sectionId/seats/available?event_id=<event_id>`

**Purpose:** Retrieve all available seats in a specific section for the event.

**Authentication:** JWT token required

**Example:** `GET /api/v1/sections/c4ccd0fb-c85f-467b-8fc1-a9fc12852ecb/seats/available?event_id=0d359479-818e-424e-87fc-632e6b3db58b`

**Headers:**

```
Authorization: Bearer <access_token>
```

**Response:**

```json
{
  "status": "success",
  "status_code": 200,
  "message": "Available seats retrieved successfully",
  "data": [
    {
      "id": "97c7ecdd-af68-4cd8-bd06-e8febd06cb08",
      "seat_number": "B1",
      "row": "B",
      "position": 1,
      "status": "AVAILABLE",
      "price": 0,
      "is_held": false
    },
    {
      "id": "c266460d-ad57-4b10-8eff-fad37734cc62",
      "seat_number": "B2",
      "row": "B",
      "position": 2,
      "status": "AVAILABLE",
      "price": 0,
      "is_held": false
    }
  ]
}
```

---

### 5. Hold Seats (Reserve Temporarily)

**Endpoint:** `POST /api/v1/seats/hold`

**Purpose:** Temporarily reserve selected seats to prevent other users from booking them during the checkout process.

**Authentication:** JWT token required

**Request:**

```json
{
  "event_id": "0d359479-818e-424e-87fc-632e6b3db58b",
  "seat_ids": [
    "97c7ecdd-af68-4cd8-bd06-e8febd06cb08",
    "c266460d-ad57-4b10-8eff-fad37734cc62",
    "ed846370-51b2-49a6-9bf0-68c7c1ff5f1c",
    "82314428-4b2a-4c77-a6a8-14ab11cc9afc",
    "80919233-307e-4049-b514-d4edaf7fe932"
  ],
  "user_id": "295fd5c1-24a8-4e7f-ab8b-87f90369c3af"
}
```

**Response:** Hold confirmation or error if seats are already held/booked

---

### 6. Get User Holds

**Endpoint:** `GET /api/v1/users/:userId/holds`

**Purpose:** Retrieve current seat holds for the authenticated user, including TTL information.

**Authentication:** JWT token required

**Example:** `GET /api/v1/users/c8d74182-7bb8-48c2-b984-b68f626bf019/holds`

**Response:**

```json
{
  "status": "success",
  "status_code": 200,
  "message": "User holds retrieved successfully",
  "data": [
    {
      "hold_id": "3fcca37e-0716-4254-8266-9a6b4ee5ec2c",
      "user_id": "c8d74182-7bb8-48c2-b984-b68f626bf019",
      "event_id": "0d359479-818e-424e-87fc-632e6b3db58b",
      "seat_ids": [
        "97c7ecdd-af68-4cd8-bd06-e8febd06cb08",
        "c266460d-ad57-4b10-8eff-fad37734cc62",
        "ed846370-51b2-49a6-9bf0-68c7c1ff5f1c",
        "82314428-4b2a-4c77-a6a8-14ab11cc9afc",
        "80919233-307e-4049-b514-d4edaf7fe932"
      ],
      "ttl_seconds": 481
    }
  ]
}
```

**Notes:**

- `ttl_seconds` shows remaining time before hold expires
- Users must complete booking before TTL expires

---

### 7. Confirm Booking & Process Payment

**Endpoint:** `POST /api/v1/booking/confirm`

**Purpose:** Convert held seats into a confirmed booking and process payment.

**Authentication:** JWT token required

**Request:**

```json
{
  "hold_id": "3fcca37e-0716-4254-8266-9a6b4ee5ec2c",
  "event_id": "0d359479-818e-424e-87fc-632e6b3db58b",
  "payment_method": "credit_card"
}
```

**Response:**

```json
{
  "data": {
    "booking_id": "96c5c807-a825-47e5-961c-42f488694bb6",
    "booking_ref": "EVT-20250914-OPSXOX",
    "status": "CONFIRMED",
    "total_price": 2500,
    "total_seats": 5,
    "version": 1,
    "seats": [
      {
        "seat_id": "97c7ecdd-af68-4cd8-bd06-e8febd06cb08",
        "section_id": "c4ccd0fb-c85f-467b-8fc1-a9fc12852ecb",
        "seat_number": "B1",
        "row": "B",
        "section_name": "General",
        "price": 500
      }
    ],
    "payment": {
      "id": "ceb16252-f178-4013-a5fa-b5fcb307ea4a",
      "amount": 2500,
      "currency": "INR",
      "status": "COMPLETED",
      "payment_method": "credit_card",
      "transaction_id": "TXN_1757796059_91E98942",
      "processed_at": "2025-09-14T02:10:59.672974+05:30"
    },
    "created_at": "2025-09-13T20:40:59.662077Z"
  },
  "message": "Booking confirmed successfully"
}
```

**Notes:**

- Booking reference `booking_ref` is generated for user identification
- Payment is processed atomically with booking confirmation
- Version field supports optimistic concurrency control

---

### 8. View Booking History

**Endpoint:** `GET /api/v1/users/bookings`

**Purpose:** Retrieve all bookings made by the authenticated user.

**Authentication:** JWT token required

**Response:**

```json
{
  "data": {
    "bookings": [
      {
        "id": "96c5c807-a825-47e5-961c-42f488694bb6",
        "user_id": "c8d74182-7bb8-48c2-b984-b68f626bf019",
        "event_id": "0d359479-818e-424e-87fc-632e6b3db58b",
        "total_seats": 5,
        "total_price": 2500,
        "status": "CONFIRMED",
        "booking_ref": "EVT-20250914-OPSXOX",
        "version": 1,
        "created_at": "2025-09-14T02:10:59.662077+05:30",
        "updated_at": "2025-09-14T02:10:59.662077+05:30",
        "seat_bookings": [
          {
            "id": "0e16375f-ebfa-49e6-bf60-c35e16f15a5e",
            "booking_id": "96c5c807-a825-47e5-961c-42f488694bb6",
            "event_id": "0d359479-818e-424e-87fc-632e6b3db58b",
            "seat_id": "97c7ecdd-af68-4cd8-bd06-e8febd06cb08",
            "section_id": "c4ccd0fb-c85f-467b-8fc1-a9fc12852ecb",
            "seat_price": 500,
            "created_at": "2025-09-14T02:10:59.664814+05:30"
          }
        ],
        "payments": [
          {
            "id": "ceb16252-f178-4013-a5fa-b5fcb307ea4a",
            "booking_id": "96c5c807-a825-47e5-961c-42f488694bb6",
            "amount": 2500,
            "currency": "INR",
            "status": "COMPLETED",
            "payment_method": "credit_card",
            "transaction_id": "TXN_1757796059_91E98942",
            "processed_at": "2025-09-14T02:10:59.672974+05:30",
            "created_at": "2025-09-14T02:10:59.666611+05:30",
            "updated_at": "2025-09-14T02:10:59.673219+05:30"
          }
        ]
      }
    ],
    "count": 1,
    "limit": 10,
    "offset": 0
  },
  "message": "Bookings retrieved successfully"
}
```

## Key Features Demonstrated

### Concurrency Control

- **Seat Holding Mechanism**: Temporary reservation prevents race conditions
- **TTL-based Holds**: Automatic expiration prevents indefinite locks
- **Version Control**: Optimistic concurrency control for booking updates

### Security

- **JWT Authentication**: Secure token-based authentication
- **User Authorization**: User-specific operations require proper authentication
- **Input Validation**: Structured request/response validation

### User Experience

- **Multi-step Flow**: Clear separation of browsing, selection, and confirmation
- **Real-time Availability**: Live seat availability checking
- **Booking References**: Human-readable booking identifiers
- **Comprehensive History**: Complete booking and payment details

### Scalability Considerations

- **Stateless Design**: JWT tokens enable horizontal scaling
- **Efficient Queries**: Section-based seat retrieval reduces database load
- **Atomic Operations**: Transaction-safe booking confirmation
- **Pagination Support**: Built-in pagination for large result sets

## Error Handling

The API implements comprehensive error handling:

- **400 Bad Request**: Invalid input data or parameters
- **401 Unauthorized**: Missing or invalid authentication
- **404 Not Found**: Requested resource doesn't exist
- **409 Conflict**: Seat already held or booked by another user
- **500 Internal Server Error**: Unexpected server errors

## Base URL

All endpoints use the base URL: `https://www.evently-api.mitshah.dev/api/v1`

## Authentication

Most endpoints require JWT authentication. Include the token in the Authorization header:

```
Authorization: Bearer <access_token>
```
