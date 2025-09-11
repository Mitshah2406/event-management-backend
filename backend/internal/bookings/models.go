package bookings

import (
	"time"

	"github.com/google/uuid"
)

// Booking defines the main booking structure
type Booking struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	UserID      uuid.UUID  `gorm:"type:uuid;index;not null" json:"user_id"`
	EventID     uuid.UUID  `gorm:"type:uuid;index;not null" json:"event_id"`
	TotalSeats  int        `gorm:"not null" json:"total_seats"`
	TotalPrice  float64    `gorm:"not null" json:"total_price"`
	Status      string     `gorm:"type:varchar(20);check:status IN ('CONFIRMED', 'CANCELLED');default:'CONFIRMED'" json:"status"`
	BookingRef  string     `gorm:"unique;not null" json:"booking_ref"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`

	// Relationships
	SeatBookings []SeatBooking `json:"seat_bookings,omitempty" gorm:"foreignKey:BookingID;constraint:OnDelete:CASCADE;"`
	Payments     []Payment     `json:"payments,omitempty" gorm:"foreignKey:BookingID;constraint:OnDelete:RESTRICT;"`
}

// SeatBooking defines the structure for individual seat bookings
type SeatBooking struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	BookingID uuid.UUID `gorm:"type:uuid;index;not null" json:"booking_id"`
	SeatID    uuid.UUID `gorm:"type:uuid;index;not null" json:"seat_id"`
	SectionID uuid.UUID `gorm:"type:uuid;not null" json:"section_id"`
	SeatPrice float64   `gorm:"not null" json:"seat_price"`
	CreatedAt time.Time `json:"created_at"`

	// Relationships
	Booking *Booking `json:"booking,omitempty" gorm:"foreignKey:BookingID;constraint:OnDelete:CASCADE;"`
	Seat    *Seat    `json:"seat,omitempty" gorm:"foreignKey:SeatID;constraint:OnDelete:RESTRICT;"`
}

// Payment defines the structure for payment tracking
type Payment struct {
	ID            uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	BookingID     uuid.UUID  `gorm:"type:uuid;index;not null" json:"booking_id"`
	Amount        float64    `gorm:"not null" json:"amount"`
	Currency      string     `gorm:"type:varchar(3);default:'USD'" json:"currency"`
	Status        string     `gorm:"type:varchar(20);check:status IN ('PENDING', 'COMPLETED', 'FAILED', 'REFUNDED');default:'PENDING'" json:"status"`
	PaymentMethod string     `gorm:"type:varchar(50)" json:"payment_method"`
	TransactionID string     `gorm:"unique" json:"transaction_id"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
	FailureReason string     `json:"failure_reason,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// Relationships
	Booking *Booking `json:"booking,omitempty" gorm:"foreignKey:BookingID;constraint:OnDelete:CASCADE;"`
}

// Forward declarations
type Seat struct {
	ID         uuid.UUID `json:"id"`
	SectionID  uuid.UUID `json:"section_id"`
	SeatNumber string    `json:"seat_number"`
	Row        string    `json:"row"`
	Position   int       `json:"position"`
	Status     string    `json:"status"`
}

// TableName sets the table name for Booking
func (Booking) TableName() string {
	return "bookings"
}

// TableName sets the table name for SeatBooking
func (SeatBooking) TableName() string {
	return "seat_bookings"
}

// TableName sets the table name for Payment
func (Payment) TableName() string {
	return "payments"
}

// BookingConfirmationRequest represents booking confirmation request
type BookingConfirmationRequest struct {
	HoldID        string `json:"hold_id" binding:"required"`
	EventID       string `json:"event_id" binding:"required,uuid"`
	PaymentMethod string `json:"payment_method" binding:"required"`
}

// BookingConfirmationResponse represents booking confirmation response
type BookingConfirmationResponse struct {
	BookingID  string           `json:"booking_id"`
	BookingRef string           `json:"booking_ref"`
	Status     string           `json:"status"`
	TotalPrice float64          `json:"total_price"`
	TotalSeats int              `json:"total_seats"`
	Seats      []BookedSeatInfo `json:"seats"`
	Payment    PaymentInfo      `json:"payment"`
	CreatedAt  time.Time        `json:"created_at"`
}

// BookedSeatInfo represents information about booked seats
type BookedSeatInfo struct {
	SeatID      string  `json:"seat_id"`
	SectionID   string  `json:"section_id"`
	SeatNumber  string  `json:"seat_number"`
	Row         string  `json:"row"`
	SectionName string  `json:"section_name"`
	Price       float64 `json:"price"`
}

// PaymentInfo represents payment information in responses
type PaymentInfo struct {
	ID            string     `json:"id"`
	Amount        float64    `json:"amount"`
	Currency      string     `json:"currency"`
	Status        string     `json:"status"`
	PaymentMethod string     `json:"payment_method"`
	TransactionID string     `json:"transaction_id"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
}

// Helper methods for booking management
func (b *Booking) IsConfirmed() bool {
	return b.Status == "CONFIRMED"
}

func (b *Booking) IsCancelled() bool {
	return b.Status == "CANCELLED"
}

func (b *Booking) Cancel() {
	b.Status = "CANCELLED"
	now := time.Now()
	b.CancelledAt = &now
	b.UpdatedAt = now
}

// Helper methods for payment management
func (p *Payment) IsPending() bool {
	return p.Status == "PENDING"
}

func (p *Payment) IsCompleted() bool {
	return p.Status == "COMPLETED"
}

func (p *Payment) IsFailed() bool {
	return p.Status == "FAILED"
}

func (p *Payment) IsRefunded() bool {
	return p.Status == "REFUNDED"
}

func (p *Payment) MarkCompleted(transactionID string) {
	p.Status = "COMPLETED"
	p.TransactionID = transactionID
	now := time.Now()
	p.ProcessedAt = &now
	p.UpdatedAt = now
}

func (p *Payment) MarkFailed(reason string) {
	p.Status = "FAILED"
	p.FailureReason = reason
	now := time.Now()
	p.ProcessedAt = &now
	p.UpdatedAt = now
}

// Convert Payment to PaymentInfo
func (p *Payment) ToPaymentInfo() PaymentInfo {
	return PaymentInfo{
		ID:            p.ID.String(),
		Amount:        p.Amount,
		Currency:      p.Currency,
		Status:        p.Status,
		PaymentMethod: p.PaymentMethod,
		TransactionID: p.TransactionID,
		ProcessedAt:   p.ProcessedAt,
	}
}
