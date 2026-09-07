package repository

import (
	"strings"
	"testing"
)

// TestStatementBuilderUsesDollarPlaceholders - проверяет, что билдер ставит плейсхолдеры вида $1.
func TestStatementBuilderUsesDollarPlaceholders(t *testing.T) {
	query, _, err := sb.Select("*").From("orders").Where("id = ?", "x").ToSql()
	if err != nil {
		t.Fatalf("build query: %v", err)
	}
	if !strings.Contains(query, "$1") {
		t.Errorf("expected $1 placeholder, got: %s", query)
	}
	if strings.Contains(query, "?") {
		t.Errorf("found ? placeholder, got: %s", query)
	}
}
