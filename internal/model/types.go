package model

// ContextKey используется для ключей контекста запроса.
type ContextKey string

const (
	CtxUser    ContextKey = "user"
	CtxSession ContextKey = "session"
	CtxCSRF    ContextKey = "csrf_token"
	CtxNonce   ContextKey = "csp_nonce"
)

// PaginationParams содержит параметры пагинации.
type PaginationParams struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// Offset возвращает смещение для SQL LIMIT/OFFSET.
func (p PaginationParams) Offset() int {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 {
		p.Limit = 30
	}
	return (p.Page - 1) * p.Limit
}

// PaginationMeta содержит мета-информацию о пагинации.
type PaginationMeta struct {
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// TotalPages вычисляет общее количество страниц.
func (m *PaginationMeta) TotalPages() int {
	if m.Limit <= 0 {
		return 0
	}
	totalPages := m.Total / m.Limit
	if m.Total%m.Limit != 0 {
		totalPages++
	}
	return totalPages
}