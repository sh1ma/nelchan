-- Migration number: 0001 	 2025-12-26T11:52:05.823Z
CREATE TABLE commands (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    author_id TEXT NOT NULL
);

CREATE TABLE dictionaries (
    id TEXT PRIMARY KEY,
    command_id TEXT NOT NULL,
    text TEXT NOT NULL,
);

CREATE TABLE codes (
    id TEXT PRIMARY KEY,
    command_id TEXT NOT NULL,
    code TEXT NOT NULL,
);