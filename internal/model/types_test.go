package model

import (
	"testing"
)

func TestPaginationParamsOffset(t *testing.T) {
	tests := []struct {
		name   string
		params PaginationParams
		want   int
	}{
		{"page 1 default limit", PaginationParams{Page: 1, Limit: 30}, 0},
		{"page 2 default limit", PaginationParams{Page: 2, Limit: 30}, 30},
		{"page 5 default limit", PaginationParams{Page: 5, Limit: 30}, 120},
		{"page 1 custom limit", PaginationParams{Page: 1, Limit: 10}, 0},
		{"page 3 custom limit", PaginationParams{Page: 3, Limit: 10}, 20},
		{"page 0 treated as 1", PaginationParams{Page: 0, Limit: 30}, 0},
		{"negative page treated as 1", PaginationParams{Page: -1, Limit: 30}, 0},
		{"zero limit defaults to 30", PaginationParams{Page: 1, Limit: 0}, 0},
		{"negative limit defaults to 30", PaginationParams{Page: 2, Limit: -5}, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.params.Offset(); got != tt.want {
				t.Errorf("PaginationParams{Page: %d, Limit: %d}.Offset() = %d, want %d",
					tt.params.Page, tt.params.Limit, got, tt.want)
			}
		})
	}
}

func TestPaginationMetaTotalPages(t *testing.T) {
	tests := []struct {
		name string
		meta PaginationMeta
		want int
	}{
		{"exact division", PaginationMeta{Total: 90, Limit: 30, Page: 1}, 3},
		{"with remainder", PaginationMeta{Total: 100, Limit: 30, Page: 1}, 4},
		{"single page", PaginationMeta{Total: 15, Limit: 30, Page: 1}, 1},
		{"zero total", PaginationMeta{Total: 0, Limit: 30, Page: 1}, 0},
		{"zero limit", PaginationMeta{Total: 100, Limit: 0, Page: 1}, 0},
		{"negative limit", PaginationMeta{Total: 100, Limit: -10, Page: 1}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.meta.TotalPages(); got != tt.want {
				t.Errorf("PaginationMeta{Total: %d, Limit: %d}.TotalPages() = %d, want %d",
					tt.meta.Total, tt.meta.Limit, got, tt.want)
			}
		})
	}
}
