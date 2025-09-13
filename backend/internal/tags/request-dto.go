package tags

type CreateTagRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Description string `json:"description" binding:"max=500"`
	Color       string `json:"color" binding:"omitempty,len=7"` // Hex color validation
}

type UpdateTagRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=2,max=100"`
	Description *string `json:"description" binding:"omitempty,max=500"`
	Color       *string `json:"color" binding:"omitempty,len=7"`
	IsActive    *bool   `json:"is_active"`
}

type TagListQuery struct {
	Page      int    `form:"page" binding:"omitempty,min=1"`
	Limit     int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Search    string `form:"search"`
	IsActive  *bool  `form:"is_active"`
	SortBy    string `form:"sort_by" binding:"omitempty,oneof=name created_at updated_at"`
	SortOrder string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}
