CREATE TABLE teams (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    team_id INT NOT NULL REFERENCES teams(id) ON DELETE CASCADE
);

CREATE INDEX idx_users_team_id ON users(team_id);