package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"evently/internal/cancellation"
	"evently/internal/events"
	"evently/internal/seats"
	"evently/internal/shared/config"
	"evently/internal/shared/database"
	"evently/internal/tags"
	"evently/internal/users"
	"evently/internal/venues"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Seeder struct {
	db *database.DB
}

func main() {
	fmt.Println("🌱 Starting Evently Database Seeder...")

	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	seeder := &Seeder{db: db}

	// Clean database
	fmt.Println("\n🧹 Cleaning database...")
	if err := seeder.CleanDatabase(); err != nil {
		log.Fatalf("Failed to clean database: %v", err)
	}
	fmt.Println("✅ Database cleaned successfully")

	// Seed data
	fmt.Println("\n🌱 Seeding database...")
	if err := seeder.SeedAll(); err != nil {
		log.Fatalf("Failed to seed database: %v", err)
	}
	fmt.Println("✅ Database seeded successfully")

	fmt.Println("\n🎉 Seeding completed! Database is ready for testing.")
}

// CleanDatabase truncates all tables in the correct order (respecting foreign key constraints)
func (s *Seeder) CleanDatabase() error {
	// Order matters due to foreign key constraints
	// Delete in reverse dependency order
	tables := []string{
		"waitlist_notifications",
		"waitlist_analytics",
		"waitlist_entries",
		"cancellations",
		"cancellation_policies",
		"payments",
		"seat_bookings",
		"bookings",
		"event_pricing",
		"event_tags",
		"seats",
		"venue_sections",
		"venue_templates",
		"events",
		"tags",
		"users",
	}

	tx := s.db.PostgreSQL.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Disable foreign key constraints temporarily
	if err := tx.Exec("SET CONSTRAINTS ALL DEFERRED").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to defer constraints: %w", err)
	}

	for _, table := range tables {
		fmt.Printf("  Truncating table: %s\n", table)
		if err := tx.Exec(fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table)).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}

	// Re-enable foreign key constraints
	if err := tx.Exec("SET CONSTRAINTS ALL IMMEDIATE").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to restore constraints: %w", err)
	}

	return tx.Commit().Error
}

// SeedAll seeds all required data
func (s *Seeder) SeedAll() error {
	ctx := context.Background()

	// Seed users first (no dependencies)
	userIDs, err := s.SeedUsers()
	if err != nil {
		return fmt.Errorf("failed to seed users: %w", err)
	}

	// Seed tags (no dependencies)
	tagIDs, err := s.SeedTags(userIDs["admin"])
	if err != nil {
		return fmt.Errorf("failed to seed tags: %w", err)
	}

	// Seed venue templates and sections
	venueTemplateIDs, err := s.SeedVenueTemplates()
	if err != nil {
		return fmt.Errorf("failed to seed venue templates: %w", err)
	}

	// Seed events
	eventIDs, err := s.SeedEvents(userIDs["admin"], venueTemplateIDs, tagIDs)
	if err != nil {
		return fmt.Errorf("failed to seed events: %w", err)
	}

	// Seed cancellation policies
	if err := s.SeedCancellationPolicies(eventIDs); err != nil {
		return fmt.Errorf("failed to seed cancellation policies: %w", err)
	}

	// Clear Redis cache to ensure fresh state
	if err := s.db.Redis.FlushDB(ctx).Err(); err != nil {
		log.Printf("Warning: Failed to clear Redis cache: %v", err)
	}

	return nil
}

// SeedUsers creates 3 users: 1 admin and 2 regular users
func (s *Seeder) SeedUsers() (map[string]uuid.UUID, error) {
	fmt.Println("  👤 Seeding users...")

	userIDs := make(map[string]uuid.UUID)

	// Hash password for all users (using "qwerty")
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("qwerty"), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	usersData := []struct {
		key       string
		firstName string
		lastName  string
		email     string
		role      users.Role
	}{
		{"admin", "Admin", "User", "admin@gmail.com", users.RoleAdmin},
		{"user1", "Mit", "Shah", "mitshah2406@gmail.com", users.RoleUser},
		{"user2", "Mit", "Shah", "mitshah2406.work@gmail.com", users.RoleUser},
	}

	for _, userData := range usersData {
		user := users.User{
			ID:        uuid.New(),
			FirstName: userData.firstName,
			LastName:  userData.lastName,
			Email:     userData.email,
			Password:  string(hashedPassword),
			Role:      userData.role,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := s.db.PostgreSQL.Create(&user).Error; err != nil {
			return nil, fmt.Errorf("failed to create user %s: %w", userData.email, err)
		}

		userIDs[userData.key] = user.ID
		fmt.Printf("    ✅ Created user: %s (%s)\n", user.Email, user.Role)
	}

	return userIDs, nil
}

// SeedTags creates event tags
func (s *Seeder) SeedTags(adminID uuid.UUID) ([]uuid.UUID, error) {
	fmt.Println("  🏷️ Seeding tags...")

	var tagIDs []uuid.UUID

	tagsData := []struct {
		name        string
		description string
		color       string
	}{
		{"Technology", "Technology and innovation events", "#3B82F6"},
		{"Music", "Musical performances and concerts", "#EF4444"},
		{"Sports", "Sporting events and competitions", "#10B981"},
		{"Business", "Business conferences and networking", "#8B5CF6"},
		{"Arts", "Art exhibitions and cultural events", "#F59E0B"},
		{"Food", "Culinary events and food festivals", "#EC4899"},
	}

	for _, tagData := range tagsData {
		tag := tags.Tag{
			ID:          uuid.New(),
			Name:        tagData.name,
			Slug:        generateSlug(tagData.name),
			Description: tagData.description,
			Color:       tagData.color,
			IsActive:    true,
			CreatedBy:   adminID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := s.db.PostgreSQL.Create(&tag).Error; err != nil {
			return nil, fmt.Errorf("failed to create tag %s: %w", tag.Name, err)
		}

		tagIDs = append(tagIDs, tag.ID)
		fmt.Printf("    ✅ Created tag: %s\n", tag.Name)
	}

	return tagIDs, nil
}

// SeedVenueTemplates creates venue templates with sections and seats
func (s *Seeder) SeedVenueTemplates() ([]uuid.UUID, error) {
	fmt.Println("  🏟️ Seeding venue templates...")

	var templateIDs []uuid.UUID

	// Venue Template 1: Small Theater
	template1 := venues.VenueTemplate{
		ID:                 uuid.New(),
		Name:               "Small Theater",
		Description:        "Intimate theater with premium and standard seating",
		DefaultRows:        2,
		DefaultSeatsPerRow: 13,
		LayoutType:         "THEATER",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := s.db.PostgreSQL.Create(&template1).Error; err != nil {
		return nil, fmt.Errorf("failed to create venue template 1: %w", err)
	}
	templateIDs = append(templateIDs, template1.ID)
	fmt.Printf("    ✅ Created venue template: %s\n", template1.Name)

	// Create sections for Small Theater
	if err := s.createTheaterSections(template1.ID); err != nil {
		return nil, fmt.Errorf("failed to create sections for Small Theater: %w", err)
	}

	// Venue Template 2: Conference Hall
	template2 := venues.VenueTemplate{
		ID:                 uuid.New(),
		Name:               "Conference Hall",
		Description:        "Professional conference hall with premium and general seating",
		DefaultRows:        2,
		DefaultSeatsPerRow: 13,
		LayoutType:         "CONFERENCE",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := s.db.PostgreSQL.Create(&template2).Error; err != nil {
		return nil, fmt.Errorf("failed to create venue template 2: %w", err)
	}
	templateIDs = append(templateIDs, template2.ID)
	fmt.Printf("    ✅ Created venue template: %s\n", template2.Name)

	// Create sections for Conference Hall
	if err := s.createConferenceHallSections(template2.ID); err != nil {
		return nil, fmt.Errorf("failed to create sections for Conference Hall: %w", err)
	}

	return templateIDs, nil
}

// createTheaterSections creates sections for the theater template
func (s *Seeder) createTheaterSections(templateID uuid.UUID) error {
	// Premium Section (A-M seats)
	premiumSection := venues.VenueSection{
		ID:          uuid.New(),
		TemplateID:  templateID,
		Name:        "Premium",
		Description: "Premium seating with the best view",
		RowStart:    "A",
		RowEnd:      "A",
		SeatsPerRow: 13, // A1-A13 (13 seats)
		TotalSeats:  13,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.db.PostgreSQL.Create(&premiumSection).Error; err != nil {
		return fmt.Errorf("failed to create premium section: %w", err)
	}
	fmt.Printf("      ✅ Created section: %s (%d seats)\n", premiumSection.Name, premiumSection.TotalSeats)

	// Create seats for premium section (A1-A13)
	if err := s.createSeatsForSection(premiumSection.ID, "A", 13); err != nil {
		return fmt.Errorf("failed to create premium seats: %w", err)
	}

	// Standard Section (N-Z seats)
	standardSection := venues.VenueSection{
		ID:          uuid.New(),
		TemplateID:  templateID,
		Name:        "Standard",
		Description: "Standard seating with good view",
		RowStart:    "B",
		RowEnd:      "B",
		SeatsPerRow: 13, // B1-B13 (13 seats)
		TotalSeats:  13,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.db.PostgreSQL.Create(&standardSection).Error; err != nil {
		return fmt.Errorf("failed to create standard section: %w", err)
	}
	fmt.Printf("      ✅ Created section: %s (%d seats)\n", standardSection.Name, standardSection.TotalSeats)

	// Create seats for standard section (B1-B13)
	if err := s.createSeatsForSection(standardSection.ID, "B", 13); err != nil {
		return fmt.Errorf("failed to create standard seats: %w", err)
	}

	return nil
}

// createConferenceHallSections creates sections for the conference hall template
func (s *Seeder) createConferenceHallSections(templateID uuid.UUID) error {
	// VIP Section
	vipSection := venues.VenueSection{
		ID:          uuid.New(),
		TemplateID:  templateID,
		Name:        "VIP",
		Description: "VIP seating with premium amenities",
		RowStart:    "A",
		RowEnd:      "A",
		SeatsPerRow: 8, // A1-A8 (8 seats)
		TotalSeats:  8,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.db.PostgreSQL.Create(&vipSection).Error; err != nil {
		return fmt.Errorf("failed to create VIP section: %w", err)
	}
	fmt.Printf("      ✅ Created section: %s (%d seats)\n", vipSection.Name, vipSection.TotalSeats)

	// Create seats for VIP section (A1-A8)
	if err := s.createSeatsForSection(vipSection.ID, "A", 8); err != nil {
		return fmt.Errorf("failed to create VIP seats: %w", err)
	}

	// General Section
	generalSection := venues.VenueSection{
		ID:          uuid.New(),
		TemplateID:  templateID,
		Name:        "General",
		Description: "General seating for all attendees",
		RowStart:    "B",
		RowEnd:      "C",
		SeatsPerRow: 10, // B1-B10, C1-C10 (20 seats total)
		TotalSeats:  20,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.db.PostgreSQL.Create(&generalSection).Error; err != nil {
		return fmt.Errorf("failed to create general section: %w", err)
	}
	fmt.Printf("      ✅ Created section: %s (%d seats)\n", generalSection.Name, generalSection.TotalSeats)

	// Create seats for general section (B1-B10, C1-C10)
	if err := s.createSeatsForSection(generalSection.ID, "B", 10); err != nil {
		return fmt.Errorf("failed to create general seats row B: %w", err)
	}
	if err := s.createSeatsForSection(generalSection.ID, "C", 10); err != nil {
		return fmt.Errorf("failed to create general seats row C: %w", err)
	}

	return nil
}

// createSeatsForSection creates individual seats for a section
func (s *Seeder) createSeatsForSection(sectionID uuid.UUID, row string, seatCount int) error {
	for i := 1; i <= seatCount; i++ {
		seat := seats.Seat{
			ID:         uuid.New(),
			SectionID:  sectionID,
			SeatNumber: fmt.Sprintf("%s%d", row, i),
			Row:        row,
			Position:   i,
			Status:     "AVAILABLE",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		if err := s.db.PostgreSQL.Create(&seat).Error; err != nil {
			return fmt.Errorf("failed to create seat %s: %w", seat.SeatNumber, err)
		}
	}

	return nil
}

// SeedEvents creates sample events
func (s *Seeder) SeedEvents(adminID uuid.UUID, venueTemplateIDs []uuid.UUID, tagIDs []uuid.UUID) ([]uuid.UUID, error) {
	fmt.Println("  🎪 Seeding events...")

	var eventIDs []uuid.UUID

	eventsData := []struct {
		name            string
		description     string
		venue           string
		venueTemplateID uuid.UUID
		basePrice       float64
		daysFromNow     int
		tagIndexes      []int
		sectionPricing  map[string]float64 // section name -> price multiplier
	}{
		{
			name:            "Tech Conference 2024",
			description:     "Annual technology conference featuring the latest innovations and industry leaders.",
			venue:           "Tech Hub Convention Center",
			venueTemplateID: venueTemplateIDs[1], // Conference Hall
			basePrice:       1500.0,
			daysFromNow:     30,
			tagIndexes:      []int{0, 3}, // Technology, Business
			sectionPricing: map[string]float64{
				"VIP":     2.0, // 3000 INR
				"General": 1.0, // 1500 INR
			},
		},
		{
			name:            "Classical Music Evening",
			description:     "An elegant evening of classical music performed by renowned musicians.",
			venue:           "Grand Opera House",
			venueTemplateID: venueTemplateIDs[0], // Small Theater
			basePrice:       800.0,
			daysFromNow:     45,
			tagIndexes:      []int{1, 4}, // Music, Arts
			sectionPricing: map[string]float64{
				"Premium":  1.8, // 1440 INR
				"Standard": 1.0, // 800 INR
			},
		},
		{
			name:            "Startup Pitch Night",
			description:     "Watch promising startups pitch their ideas to investors and industry experts.",
			venue:           "Innovation Center",
			venueTemplateID: venueTemplateIDs[1], // Conference Hall
			basePrice:       500.0,
			daysFromNow:     15,
			tagIndexes:      []int{0, 3}, // Technology, Business
			sectionPricing: map[string]float64{
				"VIP":     1.5, // 750 INR
				"General": 1.0, // 500 INR
			},
		},
		{
			name:            "Food & Wine Festival",
			description:     "A delightful festival celebrating local cuisine and fine wines.",
			venue:           "Central Park Pavilion",
			venueTemplateID: venueTemplateIDs[0], // Small Theater
			basePrice:       1200.0,
			daysFromNow:     60,
			tagIndexes:      []int{5}, // Food
			sectionPricing: map[string]float64{
				"Premium":  1.5, // 1800 INR
				"Standard": 1.0, // 1200 INR
			},
		},
		{
			name:            "Art Gallery Opening",
			description:     "Opening night of contemporary art exhibition featuring local and international artists.",
			venue:           "Modern Art Museum",
			venueTemplateID: venueTemplateIDs[0], // Small Theater
			basePrice:       600.0,
			daysFromNow:     25,
			tagIndexes:      []int{4}, // Arts
			sectionPricing: map[string]float64{
				"Premium":  1.3, // 780 INR
				"Standard": 1.0, // 600 INR
			},
		},
		{
			name:            "Sports Analytics Summit",
			description:     "Conference on the future of sports analytics and data-driven decision making.",
			venue:           "Sports Complex Conference Room",
			venueTemplateID: venueTemplateIDs[1], // Conference Hall
			basePrice:       2000.0,
			daysFromNow:     90,
			tagIndexes:      []int{2, 0, 3}, // Sports, Technology, Business
			sectionPricing: map[string]float64{
				"VIP":     2.5, // 5000 INR
				"General": 1.0, // 2000 INR
			},
		},
	}

	for _, eventData := range eventsData {
		event := events.Event{
			ID:              uuid.New(),
			Name:            eventData.name,
			Description:     eventData.description,
			Venue:           eventData.venue,
			VenueTemplateID: eventData.venueTemplateID,
			DateTime:        time.Now().AddDate(0, 0, eventData.daysFromNow),
			BasePrice:       eventData.basePrice,
			Status:          events.EventStatusPublished,
			ImageURL:        "",
			CreatedBy:       adminID,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if err := s.db.PostgreSQL.Create(&event).Error; err != nil {
			return nil, fmt.Errorf("failed to create event %s: %w", event.Name, err)
		}

		eventIDs = append(eventIDs, event.ID)
		fmt.Printf("    ✅ Created event: %s\n", event.Name)

		// Associate tags with event
		for _, tagIndex := range eventData.tagIndexes {
			if tagIndex < len(tagIDs) {
				eventTag := tags.EventTag{
					ID:        uuid.New(),
					EventID:   event.ID,
					TagID:     tagIDs[tagIndex],
					CreatedAt: time.Now(),
				}

				if err := s.db.PostgreSQL.Create(&eventTag).Error; err != nil {
					return nil, fmt.Errorf("failed to associate tag with event %s: %w", event.Name, err)
				}
			}
		}

		// Create event pricing for sections
		if err := s.createEventPricing(event.ID, eventData.venueTemplateID, eventData.sectionPricing); err != nil {
			return nil, fmt.Errorf("failed to create pricing for event %s: %w", event.Name, err)
		}
	}

	return eventIDs, nil
}

// createEventPricing creates event-specific pricing for venue sections
func (s *Seeder) createEventPricing(eventID, venueTemplateID uuid.UUID, pricingMap map[string]float64) error {
	// Get all sections for this venue template
	var sections []venues.VenueSection
	if err := s.db.PostgreSQL.Where("template_id = ?", venueTemplateID).Find(&sections).Error; err != nil {
		return fmt.Errorf("failed to fetch sections: %w", err)
	}

	for _, section := range sections {
		multiplier, exists := pricingMap[section.Name]
		if !exists {
			multiplier = 1.0 // Default multiplier
		}

		pricing := venues.EventPricing{
			ID:              uuid.New(),
			EventID:         eventID,
			SectionID:       section.ID,
			PriceMultiplier: multiplier,
			IsActive:        true,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if err := s.db.PostgreSQL.Create(&pricing).Error; err != nil {
			return fmt.Errorf("failed to create pricing for section %s: %w", section.Name, err)
		}
	}

	return nil
}

// SeedCancellationPolicies creates cancellation policies for events
func (s *Seeder) SeedCancellationPolicies(eventIDs []uuid.UUID) error {
	fmt.Println("  📋 Seeding cancellation policies...")

	cancellationPolicies := []struct {
		allowCancellation    bool
		deadlineDaysFromNow  int
		feeType              string
		feeAmount            float64
		refundProcessingDays int
	}{
		{true, -7, "PERCENTAGE", 10.0, 5}, // 10% fee, 7 days before event
		{true, -3, "FIXED", 50.0, 3},      // 50 INR fee, 3 days before event
		{true, -1, "PERCENTAGE", 25.0, 7}, // 25% fee, 1 day before event
		{false, 0, "NONE", 0.0, 0},        // No cancellation allowed
		{true, -5, "PERCENTAGE", 5.0, 5},  // 5% fee, 5 days before event
		{true, -10, "FIXED", 100.0, 10},   // 100 INR fee, 10 days before event
	}

	for i, eventID := range eventIDs {
		// Get event details to calculate deadline
		var event events.Event
		if err := s.db.PostgreSQL.First(&event, "id = ?", eventID).Error; err != nil {
			return fmt.Errorf("failed to fetch event: %w", err)
		}

		policyData := cancellationPolicies[i%len(cancellationPolicies)]

		var deadline time.Time
		if policyData.allowCancellation {
			deadline = event.DateTime.AddDate(0, 0, policyData.deadlineDaysFromNow)
		} else {
			deadline = event.DateTime // Set to event time for no cancellation
		}

		policy := cancellation.CancellationPolicy{
			ID:                   uuid.New(),
			EventID:              eventID,
			AllowCancellation:    policyData.allowCancellation,
			CancellationDeadline: deadline,
			FeeType:              policyData.feeType,
			FeeAmount:            policyData.feeAmount,
			RefundProcessingDays: policyData.refundProcessingDays,
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
		}

		if err := s.db.PostgreSQL.Create(&policy).Error; err != nil {
			return fmt.Errorf("failed to create cancellation policy for event: %w", err)
		}

		fmt.Printf("    ✅ Created cancellation policy for event (Allow: %v, Fee: %s)\n",
			policy.AllowCancellation, policy.FeeType)
	}

	return nil
}

// generateSlug creates a URL-friendly slug from a string
func generateSlug(name string) string {
	// Simple slug generation - convert to lowercase and replace spaces with hyphens
	slug := ""
	for _, r := range name {
		if r == ' ' {
			slug += "-"
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			if r >= 'A' && r <= 'Z' {
				slug += string(r + 32) // Convert to lowercase
			} else {
				slug += string(r)
			}
		}
	}
	return slug
}
