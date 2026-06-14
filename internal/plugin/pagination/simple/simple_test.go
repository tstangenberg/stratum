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
	"strings"
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/tstangenberg/stratum/internal/plugin/pagination/simple"
)

const baseQuery = "SELECT id, name FROM t ORDER BY id"

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
	q, params, err := p.ApplySQL(baseQuery, nil, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(q, "LIMIT $1 OFFSET $2") {
		t.Errorf("query = %q, want suffix LIMIT $1 OFFSET $2", q)
	}
	if len(params) != 2 {
		t.Fatalf("len(params) = %d, want 2", len(params))
	}
	if params[0] != 100 {
		t.Errorf("params[0] = %v, want 100", params[0])
	}
	if params[1] != 0 {
		t.Errorf("params[1] = %v, want 0", params[1])
	}
}

func TestApplySQL_WithLimit(t *testing.T) {
	p := simple.New()
	_, params, err := p.ApplySQL(baseQuery, nil, map[string]any{"limit": 42})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params[0] != 42 {
		t.Errorf("params[0] = %v, want 42", params[0])
	}
	if params[1] != 0 {
		t.Errorf("params[1] = %v, want 0", params[1])
	}
}

func TestApplySQL_WithLimitAndOffset(t *testing.T) {
	p := simple.New()
	_, params, err := p.ApplySQL(baseQuery, nil, map[string]any{"limit": 10, "offset": 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params[0] != 10 {
		t.Errorf("params[0] = %v, want 10", params[0])
	}
	if params[1] != 20 {
		t.Errorf("params[1] = %v, want 20", params[1])
	}
}

func TestApplySQL_LimitExceedsMax_ReturnsError(t *testing.T) {
	p := simple.New()
	_, _, err := p.ApplySQL(baseQuery, nil, map[string]any{"limit": 1001})
	if err == nil {
		t.Fatal("expected error for limit exceeding max")
	}
}

func TestApplySQL_NegativeLimit_ClampedToZero(t *testing.T) {
	p := simple.New()
	_, params, err := p.ApplySQL(baseQuery, nil, map[string]any{"limit": -5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params[0] != 0 {
		t.Errorf("params[0] = %v, want 0", params[0])
	}
}

func TestApplySQL_NegativeOffset_ClampedToZero(t *testing.T) {
	p := simple.New()
	_, params, err := p.ApplySQL(baseQuery, nil, map[string]any{"limit": 10, "offset": -3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params[1] != 0 {
		t.Errorf("params[1] = %v, want 0", params[1])
	}
}

func TestApplySQL_ExistingParams_CorrectIndex(t *testing.T) {
	p := simple.New()
	q, params, err := p.ApplySQL("SELECT id FROM t WHERE x = $1 ORDER BY id", []any{"foo"}, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(q, "LIMIT $2 OFFSET $3") {
		t.Errorf("query = %q, want suffix LIMIT $2 OFFSET $3", q)
	}
	if len(params) != 3 {
		t.Fatalf("len(params) = %d, want 3", len(params))
	}
	if params[0] != "foo" {
		t.Errorf("params[0] = %v, want foo", params[0])
	}
}

func TestNew_ReadsDefaultLimitFromEnv(t *testing.T) {
	t.Setenv("STRATUM_PLUGINS_PAGINATION_DEFAULT_LIMIT", "50")
	p := simple.New()
	_, params, err := p.ApplySQL(baseQuery, nil, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params[0] != 50 {
		t.Errorf("params[0] = %v, want 50", params[0])
	}
}

func TestApplySQL_DefaultLimitExceedsMax_ClampedToMax(t *testing.T) {
	t.Setenv("STRATUM_PLUGINS_PAGINATION_DEFAULT_LIMIT", "200")
	t.Setenv("STRATUM_PLUGINS_PAGINATION_MAX_LIMIT", "100")
	p := simple.New()
	_, params, err := p.ApplySQL(baseQuery, nil, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params[0] != 100 {
		t.Errorf("params[0] = %v, want 100", params[0])
	}
}

func TestNew_ReadsMaxLimitFromEnv(t *testing.T) {
	t.Setenv("STRATUM_PLUGINS_PAGINATION_MAX_LIMIT", "500")
	p := simple.New()
	_, _, err := p.ApplySQL(baseQuery, nil, map[string]any{"limit": 501})
	if err == nil {
		t.Fatal("expected error for limit exceeding custom max")
	}
}
