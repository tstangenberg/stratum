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

package apikey

// EnvAPIKey is the pre-shared key clients must send in the X-API-Key header.
// Default: none (auth disabled when unset)
const EnvAPIKey = "STRATUM_API_KEY"

// EnvMiddlewarePriority overrides the position of the api-key-auth middleware in the chain.
// Default: 100
const EnvMiddlewarePriority = "STRATUM_HTTP_MIDDLEWARE_API_KEY_AUTH_PRIORITY"
