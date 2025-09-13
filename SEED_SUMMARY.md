# Evently Database Seed Script - Quick Summary

## 🚀 Quick Start

```bash
# Option 1: Using Makefile (Recommended)
cd backend
make seed

# Option 2: Using helper script
cd backend
./seed.sh

# Option 3: Direct execution
cd backend
go run cmd/seed/main.go
```

## 📊 What Gets Created

### Users (3)

| Email                      | Password | Role  |
| -------------------------- | -------- | ----- |
| admin@gmail.com            | qwerty   | ADMIN |
| mitshah2406@gmail.com      | qwerty   | USER  |
| mitshah2406.work@gmail.com | qwerty   | USER  |

### Venues (2)

| Name            | Type       | Capacity | Sections                              |
| --------------- | ---------- | -------- | ------------------------------------- |
| Small Theater   | THEATER    | 26 seats | Premium (A1-A13), Standard (B1-B13)   |
| Conference Hall | CONFERENCE | 28 seats | VIP (A1-A8), General (B1-B10, C1-C10) |

### Events (6)

| Event                   | Venue           | Date     | Base Price | VIP/Premium | Standard/General |
| ----------------------- | --------------- | -------- | ---------- | ----------- | ---------------- |
| Tech Conference 2024    | Conference Hall | +30 days | ₹1,500     | ₹3,000      | ₹1,500           |
| Classical Music Evening | Small Theater   | +45 days | ₹800       | ₹1,440      | ₹800             |
| Startup Pitch Night     | Conference Hall | +15 days | ₹500       | ₹750        | ₹500             |
| Food & Wine Festival    | Small Theater   | +60 days | ₹1,200     | ₹1,800      | ₹1,200           |
| Art Gallery Opening     | Small Theater   | +25 days | ₹600       | ₹780        | ₹600             |
| Sports Analytics Summit | Conference Hall | +90 days | ₹2,000     | ₹5,000      | ₹2,000           |

### Cancellation Policies (6)

- Various types: No cancellation, Fixed fees (₹50-₹100), Percentage fees (5%-25%)
- Different deadlines: 1-10 days before event

## 🧪 Testing Scenarios

### 1. Authentication Flow

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

# Get specific event with venue details
curl http://localhost:8080/api/v1/events/{event_id}/venue
```

### 3. Booking Flow Test

1. Login as user (mitshah2406@gmail.com)
2. Select event with available seats
3. Choose seats from Premium/VIP or Standard/General
4. Create booking and verify pricing
5. Test cancellation (if policy allows)

### 4. Waitlist Test

1. Book all seats for an event (26-28 seats max)
2. Login as different user
3. Try to book (should be prompted for waitlist)
4. Cancel a booking to trigger waitlist processing

### 5. Admin Functions

1. Login as admin (admin@gmail.com)
2. View analytics and reports
3. Manage events and venues
4. View all bookings and cancellations

## ⚠️ Important Notes

- **Data Reset**: Script completely cleans database before seeding
- **Small Capacity**: Venues have 26-28 seats for easy manual testing
- **Future Dates**: All events are scheduled for future dates
- **Same Password**: All users use `qwerty` for convenience
- **Realistic Pricing**: Different price multipliers per section

## 🔧 Prerequisites

- PostgreSQL running and accessible
- Redis running and accessible
- Environment variables configured
- Go 1.19+ installed

## 📁 Files Created

```
backend/
├── cmd/seed/main.go         # Main seeder script
├── seed.sh                  # Helper bash script
├── Makefile                 # Updated with 'seed' target
└── SEED_README.md           # Detailed documentation
```

## 🆘 Troubleshooting

| Issue                      | Solution                                                        |
| -------------------------- | --------------------------------------------------------------- |
| Database connection failed | Check DB environment variables and ensure PostgreSQL is running |
| Redis connection failed    | Ensure Redis is running on specified host:port                  |
| Migration errors           | Run database migrations first                                   |
| Permission denied          | Check database user permissions                                 |
| Build errors               | Ensure Go dependencies are installed (`go mod tidy`)            |

---

**Ready to test your booking, cancellation, and waitlist features!** 🎉
