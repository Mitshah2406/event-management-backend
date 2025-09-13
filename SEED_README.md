# Database Seed Script

This seed script populates the Evently database with sample data for testing booking, cancellation, and waitlist functionality.

## What it does

### 🧹 Database Cleanup

- Truncates all tables in the correct order (respecting foreign key constraints)
- Clears Redis cache for a fresh state

### 🌱 Sample Data Creation

#### 👤 Users (3 total)

- **Admin User**: admin@gmail.com (Role: ADMIN)
- **User 1**: mitshah2406@gmail.com (Role: USER)
- **User 2**: mitshah2406.work@gmail.com (Role: USER)
- **Password**: `qwerty` (for all users)

#### 🏟️ Venue Templates (2 total)

**1. Small Theater**

- Layout: THEATER
- **Premium Section (Row A)**: 13 seats (A1-A13)
- **Standard Section (Row B)**: 13 seats (B1-B13)
- **Total Capacity**: 26 seats

**2. Conference Hall**

- Layout: CONFERENCE
- **VIP Section (Row A)**: 8 seats (A1-A8)
- **General Section (Rows B-C)**: 20 seats (B1-B10, C1-C10)
- **Total Capacity**: 28 seats

#### 🎪 Events (6 total)

All events are scheduled for future dates with different pricing structures:

1. **Tech Conference 2024** (Conference Hall)

   - Date: +30 days from now
   - Base Price: ₹1,500
   - VIP: ₹3,000 (2x multiplier)
   - General: ₹1,500 (1x multiplier)

2. **Classical Music Evening** (Small Theater)

   - Date: +45 days from now
   - Base Price: ₹800
   - Premium: ₹1,440 (1.8x multiplier)
   - Standard: ₹800 (1x multiplier)

3. **Startup Pitch Night** (Conference Hall)

   - Date: +15 days from now
   - Base Price: ₹500
   - VIP: ₹750 (1.5x multiplier)
   - General: ₹500 (1x multiplier)

4. **Food & Wine Festival** (Small Theater)

   - Date: +60 days from now
   - Base Price: ₹1,200
   - Premium: ₹1,800 (1.5x multiplier)
   - Standard: ₹1,200 (1x multiplier)

5. **Art Gallery Opening** (Small Theater)

   - Date: +25 days from now
   - Base Price: ₹600
   - Premium: ₹780 (1.3x multiplier)
   - Standard: ₹600 (1x multiplier)

6. **Sports Analytics Summit** (Conference Hall)
   - Date: +90 days from now
   - Base Price: ₹2,000
   - VIP: ₹5,000 (2.5x multiplier)
   - General: ₹2,000 (1x multiplier)

#### 📋 Cancellation Policies

Each event has different cancellation policies for testing:

- **Policy Types**: NONE, FIXED fee, PERCENTAGE fee
- **Deadlines**: Various days before event (1-10 days)
- **Fee Amounts**: 5%-25% or fixed amounts (₹50-₹100)

#### 🏷️ Tags

- Technology, Music, Sports, Business, Arts, Food
- Events are tagged appropriately for filtering tests

## Usage

### Option 1: Using Makefile (Recommended)

```bash
# From the backend directory
make seed
```

This command will:

- Ask for confirmation before cleaning the database
- Show progress during seeding
- Display a summary of created data
- Show login credentials

### Option 2: Direct Execution

```bash
# From the backend directory
go run cmd/seed/main.go
```

## Prerequisites

- Go 1.19+
- PostgreSQL database running and accessible
- Redis server running and accessible
- Proper environment variables set (DB connection, Redis, etc.)

## Environment Variables

Make sure these are set (usually in `.env` file):

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=evently_db
DB_USER=evently_user
DB_PASSWORD=evently_password
DB_SSLMODE=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
```

## Testing After Seeding

### 1. User Authentication

```bash
# Login as admin
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@gmail.com", "password": "qwerty"}'
```

### 2. Browse Events

```bash
# Get all events
curl http://localhost:8080/api/v1/events
```

### 3. Check Venue Details

```bash
# Get venue details for an event
curl http://localhost:8080/api/v1/events/{event_id}/venue
```

### 4. Test Booking Flow

1. Login as a user
2. Select seats from an event
3. Create a booking
4. Test cancellation (if policy allows)

### 5. Test Waitlist

1. Book all seats for an event
2. Try to join waitlist as another user
3. Cancel a booking to trigger waitlist notifications

## Manual Testing Scenarios

With the seeded data, you can manually test:

1. **Booking Different Sections**: Try booking premium vs standard seats
2. **Price Calculations**: Verify prices match section multipliers
3. **Cancellation Policies**: Test different policy types (percentage, fixed, none)
4. **Waitlist Flow**: Fill events and test waitlist functionality
5. **Admin Functions**: Use admin account for analytics and management
6. **User Management**: Test different user roles and permissions

## Notes

- **Seat Count**: Kept small (26-28 seats per venue) for easy manual testing
- **Passwords**: All users have the same password (`qwerty`) for testing convenience
- **Future Dates**: All events are scheduled in the future for realistic testing
- **Pricing**: Realistic pricing in INR with different multipliers per section
- **Data Safety**: Always cleans database completely before seeding

## Troubleshooting

### Database Connection Issues

- Ensure PostgreSQL is running
- Check environment variables
- Verify database exists and user has permissions

### Redis Connection Issues

- Ensure Redis is running
- Check Redis connection settings
- Verify Redis is accessible on specified port

### Migration Issues

- Ensure database migrations have been run
- Check if all required tables exist
- Verify foreign key constraints are properly set up

## Files Structure

```
backend/
├── cmd/
│   └── seed/
│       └── main.go          # Main seed script
├── internal/
│   └── shared/
│       ├── config/          # Configuration loading
│       └── database/        # Database connection
└── Makefile                 # Contains 'seed' target
```
