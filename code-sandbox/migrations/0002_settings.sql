-- Migration number: 0002 	 2025-12-29
-- Settings table for storing bot configuration
CREATE TABLE settings (
    mention_command TEXT
);

-- Insert initial row (single row table)
INSERT INTO settings (mention_command) VALUES (NULL);
