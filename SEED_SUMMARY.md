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
