package dto

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error" example:"Invalid request"`
	Message string `json:"message" example:"The request body is invalid"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
}

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Page     int `form:"page" example:"1"`
	PageSize int `form:"page_size" example:"10"`
}

// GetOffset calculates the offset for pagination
func (p *PaginationParams) GetOffset() int {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 10
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	return (p.Page - 1) * p.PageSize
}

// GetLimit returns the page size
func (p *PaginationParams) GetLimit() int {
	if p.PageSize < 1 {
		p.PageSize = 10
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	return p.PageSize
}

// CalculateTotalPages calculates total pages from total records
func CalculateTotalPages(total int64, pageSize int) int {
	if pageSize == 0 {
		return 0
	}
	pages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		pages++
	}
	return pages
}
