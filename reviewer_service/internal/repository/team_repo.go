package repository

import (
	"database/sql"
	"reviewer_service/internal/domain"
)

type TeamRepository interface {
	Create(name string) (int64, error)
	GetByName(name string) (*domain.Team, error)
	GetByID(id int64) (*domain.Team, error)
	Exists(name string) (bool, error)
}

type PostgresTeamRepository struct {
	db *sql.DB
}

func NewTeamRepository(db *sql.DB) *PostgresTeamRepository {
	return &PostgresTeamRepository{db: db}
}

func (r *PostgresTeamRepository) Exists(name string) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM teams WHERE name = $1)", name).Scan(&exists)
	return exists, err
}

func (r *PostgresTeamRepository) Create(name string) (int64, error) {
	var id int64
	err := r.db.QueryRow("INSERT INTO teams (name) VALUES ($1) RETURNING id", name).Scan(&id)
	return id, err
}

func (r *PostgresTeamRepository) GetByName(name string) (*domain.Team, error) {
	var team domain.Team
	err := r.db.QueryRow("SELECT id, name FROM teams WHERE name = $1", name).Scan(&team.ID, &team.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	rows, err := r.db.Query("SELECT id, username, is_active FROM users WHERE team_id = $1", team.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var user domain.User
		err := rows.Scan(&user.ID, &user.Username, &user.IsActive)
		if err != nil {
			return nil, err
		}
		user.TeamID = team.ID
		team.Members = append(team.Members, user)
	}

	return &team, nil
}

func (r *PostgresTeamRepository) GetByID(id int64) (*domain.Team, error) {
	var team domain.Team
	err := r.db.QueryRow("SELECT id, name FROM teams WHERE id = $1", id).Scan(&team.ID, &team.Name)
	if err != nil {
		return nil, err
	}
	return &team, nil
}
