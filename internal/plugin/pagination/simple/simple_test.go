// Copyright (C) 2026 Thorben Stangenberg
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package simple_test

import (
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/tstangenberg/stratum/internal/plugin/pagination/simple"
)

func TestName(t *testing.T) {
	p := simple.New()
	if p.Name() != "pagination" {
		t.Errorf("Name() = %q, want %q", p.Name(), "pagination")
	}
}

func TestArguments_ContainsLimitAndOffset(t *testing.T) {
	p := simple.New()
	args := p.Arguments(graphql.Int)
	if _, ok := args["limit"]; !ok {
		t.Error("Arguments() missing 'limit'")
	}
	if _, ok := args["offset"]; !ok {
		t.Error("Arguments() missing 'offset'")
	}
}

func TestApplySQL_NoArgs_UsesDefault(t *testing.T) {
	p := simple.New()
	limit, offset, err := p.ApplySQL(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if limit != 100 {
		t.Errorf("limit = %d, want 100", limit)
	}
	if offset != 0 {
		t.Errorf("offset = %d, want 0", offset)
	}
}

func TestApplySQL_WithLimit(t *testing.T) {
	p := simple.New()
	limit, offset, err := p.ApplySQL(map[string]any{"limit": 42})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if limit != 42 {
		t.Errorf("limit = %d, want 42", limit)
	}
	if offset != 0 {
		t.Errorf("offset = %d, want 0", offset)
	}
}

func TestApplySQL_WithLimitAndOffset(t *testing.T) {
	p := simple.New()
	limit, offset, err := p.ApplySQL(map[string]any{"limit": 10, "offset": 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if limit != 10 {
		t.Errorf("limit = %d, want 10", limit)
	}
	if offset != 20 {
		t.Errorf("offset = %d, want 20", offset)
	}
}

func TestApplySQL_LimitExceedsMax_ReturnsError(t *testing.T) {
	p := simple.New()
	_, _, err := p.ApplySQL(map[string]any{"limit": 1001})
	if err == nil {
		t.Fatal("expected error for limit exceeding max")
	}
}

func TestApplySQL_NegativeLimit_ClampedToZero(t *testing.T) {
	p := simple.New()
	limit, _, err := p.ApplySQL(map[string]any{"limit": -5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if limit != 0 {
		t.Errorf("limit = %d, want 0", limit)
	}
}

func TestApplySQL_NegativeOffset_ClampedToZero(t *testing.T) {
	p := simple.New()
	_, offset, err := p.ApplySQL(map[string]any{"limit": 10, "offset": -3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if offset != 0 {
		t.Errorf("offset = %d, want 0", offset)
	}
}

func TestNew_ReadsDefaultLimitFromEnv(t *testing.T) {
	t.Setenv("STRATUM_PLUGINS_PAGINATION_DEFAULT_LIMIT", "50")
	p := simple.New()
	limit, _, err := p.ApplySQL(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if limit != 50 {
		t.Errorf("limit = %d, want 50", limit)
	}
}

func TestApplySQL_DefaultLimitExceedsMax_ClampedToMax(t *testing.T) {
	t.Setenv("STRATUM_PLUGINS_PAGINATION_DEFAULT_LIMIT", "200")
	t.Setenv("STRATUM_PLUGINS_PAGINATION_MAX_LIMIT", "100")
	p := simple.New()
	limit, _, err := p.ApplySQL(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if limit != 100 {
		t.Errorf("limit = %d, want 100", limit)
	}
}

func TestNew_ReadsMaxLimitFromEnv(t *testing.T) {
	t.Setenv("STRATUM_PLUGINS_PAGINATION_MAX_LIMIT", "500")
	p := simple.New()
	_, _, err := p.ApplySQL(map[string]any{"limit": 501})
	if err == nil {
		t.Fatal("expected error for limit exceeding custom max")
	}
}
