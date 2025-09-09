package users

import "time"

type User struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	FirstName string    `json:"first_name" gorm:"not null"`
	LastName  string    `json:"last_name" gorm:"not null"`
	Password  string    `json:"-" gorm:"not null"` // hide in json
	Role      Role      `json:"role" gorm:"not null;default:'USER'"`
	Email     string    `json:"email" gorm:"uniqueIndex;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func IsValidRole(role string) bool {
	switch role {
	case string(RoleUser), string(RoleAdmin):
		return true
	default:
		return false
	}
}
