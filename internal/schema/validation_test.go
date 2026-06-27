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

package schema

import (
	"errors"
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2/gqlerror"
)

func TestToValidationError_NonGQLError(t *testing.T) {
	ve := toValidationError(errors.New("something unexpected"))
	if !strings.Contains(ve.Msg, "something unexpected") {
		t.Errorf("expected message to contain original error, got %q", ve.Msg)
	}
	if len(ve.Details) != 0 {
		t.Errorf("expected no details for non-gql error, got %d", len(ve.Details))
	}
}

func TestToValidationError_GQLList_AllErrorsCaptured(t *testing.T) {
	list := gqlerror.List{
		{Message: "first error", Locations: []gqlerror.Location{{Line: 1, Column: 5}}},
		{Message: "second error", Locations: []gqlerror.Location{{Line: 2, Column: 10}}},
	}
	ve := toValidationError(list)
	if len(ve.Details) != 2 {
		t.Fatalf("expected 2 details, got %d", len(ve.Details))
	}
	if ve.Details[0].Message != "first error" || ve.Details[0].Line != 1 {
		t.Errorf("unexpected first detail: %+v", ve.Details[0])
	}
	if ve.Details[1].Message != "second error" || ve.Details[1].Line != 2 {
		t.Errorf("unexpected second detail: %+v", ve.Details[1])
	}
}

func TestToValidationError_GQLList_NoLocation(t *testing.T) {
	list := gqlerror.List{
		{Message: "error without location"},
	}
	ve := toValidationError(list)
	if len(ve.Details) != 1 {
		t.Fatalf("expected 1 detail, got %d", len(ve.Details))
	}
	if ve.Details[0].Message != "error without location" {
		t.Errorf("unexpected detail message: %q", ve.Details[0].Message)
	}
	if ve.Details[0].Line != 0 || ve.Details[0].Column != 0 {
		t.Errorf("expected zero line/column for no-location entry, got %+v", ve.Details[0])
	}
}
