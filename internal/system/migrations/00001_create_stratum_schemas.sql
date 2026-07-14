-- +goose Up
CREATE TABLE stratum_system.stratum_schemas (
    name TEXT PRIMARY KEY,
    sdl TEXT NOT NULL,
    version INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE stratum_system.stratum_schemas;
