package venues

type SeatHoldRequest struct {
	EventID string   `json:"event_id" binding:"required,uuid"`
	SeatIDs []string `json:"seat_ids" binding:"required,min=1,max=10"`
	UserID  string   `json:"user_id" binding:"required,uuid"`
}

type CreateTemplateRequest struct {
	Name               string `json:"name" binding:"required,min=3,max=255"`
	Description        string `json:"description" binding:"max=1000"`
	DefaultRows        int    `json:"default_rows" binding:"required,min=1,max=50"`
	DefaultSeatsPerRow int    `json:"default_seats_per_row" binding:"required,min=1,max=100"`
	LayoutType         string `json:"layout_type" binding:"required,oneof=THEATER STADIUM CONFERENCE GENERAL"`
}

type UpdateTemplateRequest struct {
	Name               *string `json:"name" binding:"omitempty,min=3,max=255"`
	Description        *string `json:"description" binding:"omitempty,max=1000"`
	DefaultRows        *int    `json:"default_rows" binding:"omitempty,min=1,max=50"`
	DefaultSeatsPerRow *int    `json:"default_seats_per_row" binding:"omitempty,min=1,max=100"`
	LayoutType         *string `json:"layout_type" binding:"omitempty,oneof=THEATER STADIUM CONFERENCE GENERAL"`
}

type CreateSectionRequest struct {
	TemplateID  string `json:"template_id" binding:"required,uuid"`
	Name        string `json:"name" binding:"required,min=1,max=255"`
	Description string `json:"description" binding:"omitempty,max=500"`
	RowStart    string `json:"row_start" binding:"max=10"`
	RowEnd      string `json:"row_end" binding:"max=10"`
	SeatsPerRow int    `json:"seats_per_row" binding:"required,min=1,max=100"`
	TotalSeats  int    `json:"total_seats" binding:"required,min=1"`
}

type UpdateSectionRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=255"`
	Description *string `json:"description" binding:"omitempty,max=500"`
	RowStart    *string `json:"row_start" binding:"omitempty,max=10"`
	RowEnd      *string `json:"row_end" binding:"omitempty,max=10"`
	SeatsPerRow *int    `json:"seats_per_row" binding:"omitempty,min=1,max=100"`
	TotalSeats  *int    `json:"total_seats" binding:"omitempty,min=1"`
}

type CreateEventPricingRequest struct {
	EventID         string  `json:"event_id" binding:"required,uuid"`
	SectionID       string  `json:"section_id" binding:"required,uuid"`
	PriceMultiplier float64 `json:"price_multiplier" binding:"required,min=0.1,max=10"`
}

type UpdateEventPricingRequest struct {
	PriceMultiplier *float64 `json:"price_multiplier" binding:"omitempty,min=0.1,max=10"`
	IsActive        *bool    `json:"is_active"`
}
