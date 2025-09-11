package cancellation

import (
	"time"

	"github.com/google/uuid"
)

// CancellationPolicy defines the cancellation policy for events
type CancellationPolicy struct {
	ID                   uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	EventID              uuid.UUID `gorm:"type:uuid;unique;not null" json:"event_id"`
	AllowCancellation    bool      `gorm:"default:true" json:"allow_cancellation"`
	CancellationDeadline time.Time `json:"cancellation_deadline"`
	FeeType              string    `gorm:"type:varchar(20);check:fee_type IN ('NONE', 'FIXED', 'PERCENTAGE');default:'NONE'" json:"fee_type"`
	FeeAmount            float64   `gorm:"default:0" json:"fee_amount"`
	RefundProcessingDays int       `gorm:"default:5" json:"refund_processing_days"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// Cancellation defines the structure for booking cancellations
type Cancellation struct {
	ID              uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	BookingID       uuid.UUID  `gorm:"type:uuid;unique;not null" json:"booking_id"`
	RequestedAt     time.Time  `json:"requested_at"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`
	CancellationFee float64    `gorm:"default:0" json:"cancellation_fee"`
	RefundAmount    float64    `gorm:"default:0" json:"refund_amount"`
	Reason          string     `json:"reason"`
	Status          string     `gorm:"type:varchar(20);check:status IN ('PENDING', 'APPROVED', 'PROCESSED', 'REJECTED');default:'PENDING'" json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// TableName sets the table name for CancellationPolicy
func (CancellationPolicy) TableName() string {
	return "cancellation_policies"
}

// TableName sets the table name for Cancellation
func (Cancellation) TableName() string {
	return "cancellations"
}
