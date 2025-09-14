# 🎬 Evently Backend Video Recording Script

## Duration: 5-7 minutes

---

## **INTRODUCTION (30-45 seconds)**

### Opening Line:

> "Hi, I'm Mit Shah, and I've built Evently - a scalable event booking platform that handles high-concurrency ticket booking while preventing overselling and managing complex waitlist operations."

### Quick Overview:

> "Today I'll demonstrate the complete user journey from booking to cancellation, show you the automated waitlist and email notification system, explain the system architecture, and discuss the key technical challenges I solved."

### Transition:

> "Let's start with a live demonstration."

---

## **DEMO FLOW 1: SUCCESSFUL BOOKING (1.5-2 minutes)**

### Setup Context:

> "First, let me show you a successful booking flow. I have User1 who wants to book tickets for an upcoming concert."

### API Calls to Demonstrate:

#### 1. Browse Events

```
GET /api/v1/events
```

**Script:**

> "User1 starts by browsing available events. As you can see, we have several upcoming events with different venues and pricing tiers."

#### 2. User Login

```
POST /api/v1/auth/login
{
    "email": "mitshah2406@gmail.com",
    "password": "qwerty"
}
```

**Script:**

> "First, User1 needs to authenticate. Our JWT-based authentication system provides secure access tokens."

#### 3. View Event Details & Venue Layout

```
GET /api/v1/events/{event_id}/venue-layout
```

**Script:**

> "Now User1 can see the detailed venue layout with seat availability. Notice how seats are organized by sections with real-time availability status."

#### 4. Hold Seats (Critical Concurrency Point)

```
POST /api/v1/seats/hold
{
    "event_id": "event_123",
    "seat_ids": ["seat_1", "seat_2"],
    "section_id": "section_A"
}
```

**Script:**

> "Here's where the magic happens - User1 selects seats and we immediately place a Redis-based atomic lock on these seats. This prevents other users from booking the same seats while User1 completes their purchase."

#### 5. Confirm Booking

```
POST /api/v1/bookings/confirm
{
    "hold_id": "hold_abc123",
    "payment_details": { ... }
}
```

**Script:**

> "User1 confirms the booking with payment details. Our system processes this atomically - either the entire booking succeeds, or it fails completely to maintain data consistency."

#### 6. Show Booking Confirmation

```
GET /api/v1/bookings/me
```

**Script:**

> "Perfect! User1 now has a confirmed booking. Notice how the seat status has been updated in real-time and other users can no longer see these seats as available."

---

## **DEMO FLOW 2: WAITLIST & EMAIL NOTIFICATION (2-2.5 minutes)**

### Setup Context:

> "Now let me demonstrate our waitlist system. User2 wants to book the same event, but it's now sold out."

### API Calls to Demonstrate:

#### 1. User2 Login

```
POST /api/v1/auth/login
{
    "email": "mitshah2406.work@gmail.com",
    "password": "qwerty"
}
```

#### 2. Try to Book Sold-Out Event

```
GET /api/v1/events/{event_id}/seats/available
```

**Script:**

> "User2 tries to view available seats, but the event is now sold out. Let's see what happens next."

#### 3. Join Waitlist

```
POST /api/v1/waitlist/join
{
    "event_id": "event_123",
    "quantity": 2,
    "preferences": {
        "section_id": "section_A",
        "max_price": 150
    }
}
```

**Script:**

> "User2 joins the waitlist with specific preferences - 2 seats in Section A with a maximum budget of $150. Our system queues this request with FIFO ordering."

#### 4. Show Waitlist Status

```
GET /api/v1/waitlist/status/{event_id}
```

**Script:**

> "User2 can check their position in the waitlist. They're currently #1 in line for their preferred section."

#### 5. User1 Cancels Booking (The Key Moment!)

```
DELETE /api/v1/bookings/{booking_id}/cancel
```

**Script:**

> "Now here's where our automated system shines - User1 decides to cancel their booking. Watch what happens behind the scenes..."

#### 6. Show Email Notification Trigger

**Script:**

> "As soon as User1's cancellation is processed, our system automatically:
>
> 1. Releases the seat locks in Redis
> 2. Checks the waitlist for matching preferences
> 3. Finds User2 matches the criteria
> 4. Sends an immediate email notification to User2
>
> Let me show you the email that was just sent..."

**Show Email in Browser/Mail Client:**

> "Here's the actual email User2 received - it includes the event details, available seats, and a time-limited booking link. User2 has 15 minutes to complete their booking before the opportunity goes to the next person in line."

#### 7. User2 Books from Waitlist

```
POST /api/v1/seats/hold
// ... followed by booking confirmation
```

**Script:**

> "User2 clicks the link in the email and successfully books the seats that just became available. The entire process from cancellation to new booking took less than 30 seconds."

---

## **ER DIAGRAM EXPLANATION (1 minute)**

### Show ER Diagram

**Script:**

> "Let me explain the database design that makes this all possible. Here's our Entity-Relationship diagram."

### Key Relationships to Highlight:

> "The core entities are:
>
> **Users** - store authentication and profile data
> **Events** - connected to venue templates for reusable layouts  
> **Venues & Sections** - hierarchical structure for flexible seating
> **Seats** - individual seats with coordinates and pricing
> **Bookings** - link users to specific seats for events
> **Waitlist** - FIFO queue with user preferences
> **Notifications** - track email delivery status
>
> The key design decision was making Venues reusable templates - the same venue can host multiple events with different pricing and availability."

### Data Integrity Points:

> "Notice the foreign key constraints ensuring referential integrity - a booking can't exist without valid users, events, and seats. Cancellations cascade properly to update seat availability and trigger waitlist processing."

---

## **ARCHITECTURE FLOW EXPLANATIONS (1.5 minutes)**

### Flow 1: Booking Success Flow

**Script:**

> "Let me explain three critical architectural flows. First, the Booking Success Flow:
>
> 1. **Seat Selection** - User chooses seats
> 2. **Redis Lock** - Atomic seat locking prevents double-booking
> 3. **Hold Creation** - Temporary reservation with TTL
> 4. **Payment Processing** - External payment validation
> 5. **Atomic Confirmation** - Database transaction commits booking
> 6. **Lock Release** - Redis locks are cleaned up
>
> The critical insight here is using Redis for distributed locking while PostgreSQL handles the permanent state."

### Flow 2: Cancellation Flow

**Script:**

> "Second, the Cancellation Flow:
>
> 1. **Cancellation Request** - User initiates cancellation
> 2. **Validation** - Check cancellation policy and timing
> 3. **Database Update** - Mark booking as cancelled
> 4. **Seat Release** - Update seat availability
> 5. **Waitlist Trigger** - Async job processes waitlist queue
> 6. **Email Notification** - Notify eligible waitlist users
>
> This flow uses async processing to ensure cancellations are fast while waitlist processing happens in the background."

### Flow 3: Waitlist Queue Logic

**Script:**

> "Third, the Waitlist Queue Logic:
>
> 1. **Queue Management** - FIFO ordering with preference matching
> 2. **Seat Matching** - Algorithm matches available seats to user preferences
> 3. **Notification Delivery** - Reliable email delivery with retry mechanisms
> 4. **Time-Limited Offers** - 15-minute booking windows prevent stale reservations
> 5. **Cascade Processing** - If User2 doesn't respond, offer goes to User3
>
> The system handles complex scenarios like partial matches and priority-based queuing."

---

## **CHALLENGES & SOLUTIONS (1 minute)**

### Challenge 1: Race Conditions

**Script:**

> "The biggest challenge was preventing race conditions during high-concurrency booking periods. My solution uses Redis atomic operations with distributed locks. When multiple users try to book the same seat simultaneously, only one succeeds - the others receive immediate feedback to select different seats."

### Challenge 2: Waitlist Queue Management

**Script:**

> "Managing fair and efficient waitlist processing was complex. I implemented a FIFO queue with intelligent preference matching. The system can handle scenarios like: 'User wants 2 seats in Section A, but only 1 becomes available' - it keeps them in queue for a better match."

### Challenge 3: Email Reliability

**Script:**

> "Email notifications needed to be fast and reliable. I built an async notification service with retry mechanisms, delivery tracking, and fallback options. If an email fails, the system automatically retries and can escalate to the next waitlist user."

### Challenge 4: Data Consistency

**Script:**

> "Maintaining data consistency across booking, cancellation, and waitlist operations required careful transaction management. I use database transactions for critical operations and eventual consistency for non-critical updates like analytics."

---

## **CONCLUSION (30-45 seconds)**

### Summary:

**Script:**

> "To summarize, I've built a production-ready event booking platform that handles:
>
> - High-concurrency seat booking with zero overselling
> - Automated waitlist management with intelligent matching
> - Real-time email notifications
> - Scalable architecture with Redis caching and async processing
> - Comprehensive analytics for event organizers"

### Creative Features Highlight:

**Script:**

> "Some unique features I added include seat-level booking with venue templates, intelligent waitlist preference matching, and detailed analytics dashboards for event organizers."

### Closing:

**Script:**

> "The system is deployed and fully documented with comprehensive API collections. Thank you for watching, and I'm excited to discuss any questions about the technical implementation!"

---

## **🎯 RECORDING TIPS**

### Before Recording:

- [ ] Start backend services: `make docker-up && make run`
- [ ] Seed database: `make seed`
- [ ] Test all API calls in Postman
- [ ] Have ER diagram and flow diagrams ready
- [ ] Set up email client to show notifications
- [ ] Clear browser cache and organize tabs

### During Recording:

- [ ] Speak clearly and at moderate pace
- [ ] Show Postman requests and responses clearly
- [ ] Highlight important JSON fields
- [ ] Point out status codes and error handling
- [ ] Keep timing tight - practice transitions

### Technical Setup:

- [ ] Screen recording at 1080p minimum
- [ ] Good microphone quality
- [ ] Stable internet for email demonstrations
- [ ] Backup plans for any technical failures

---

## **🎬 POSTMAN COLLECTION ORDER**

### Folder 1: User1 Booking Flow

1. POST /auth/login (User1)
2. GET /events
3. GET /events/{id}/venue-layout
4. POST /seats/hold
5. POST /bookings/confirm
6. GET /bookings/me

### Folder 2: User2 Waitlist Flow

1. POST /auth/login (User2)
2. GET /events/{id}/seats/available
3. POST /waitlist/join
4. GET /waitlist/status/{event_id}

### Folder 3: Cancellation & Email

1. DELETE /bookings/{id}/cancel (as User1)
2. [Show Email Notification]
3. POST /seats/hold (as User2 from email link)
4. POST /bookings/confirm (User2)

This script should give you a solid 5-7 minute video that covers all the requirements while showcasing the technical depth and real-world applicability of your system!
