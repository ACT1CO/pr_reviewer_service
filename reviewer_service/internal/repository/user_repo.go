package repository

import (
	"database/sql"
	"fmt"
	"log"
	"reviewer_service/internal/domain"
	"strconv"
	"strings"
)

type UserRepository interface {
	UpsertMany(users []domain.User) error
	GetActiveUsersInTeamExcluding(teamID int64, excludeUserID string) ([]domain.User, error)
	GetTeamIDByUserID(userID string) (int64, error)
	GetTeamByUserID(userID string) (*domain.Team, error)
	DeactivateUsers(userIDs []string) error
	SetIsActive(userID string, isActive bool) error
	GetUserByID(userID string) (*domain.User, error)
}

type PostgresUserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) UpsertMany(users []domain.User) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			log.Printf("Failed to rollback transaction: %v", err)
		}
	}()

	stmt, err := tx.Prepare("INSERT INTO users (id, username, is_active, team_id) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO UPDATE SET username = EXCLUDED.username, is_active = EXCLUDED.is_active, team_id = EXCLUDED.team_id")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, u := range users {
		_, err := stmt.Exec(u.ID, u.Username, u.IsActive, u.TeamID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PostgresUserRepository) GetActiveUsersInTeamExcluding(teamID int64, excludeUserID string) ([]domain.User, error) {
	query := `
		SELECT id, username, is_active, team_id
		FROM users
		WHERE team_id = $1 AND is_active = true AND id != $2
	`
	rows, err := r.db.Query(query, teamID, excludeUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.IsActive, &u.TeamID); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *PostgresUserRepository) GetTeamIDByUserID(userID string) (int64, error) {
	var teamID int64
	err := r.db.QueryRow("SELECT team_id FROM users WHERE id = $1", userID).Scan(&teamID)
	if err != nil {
		return 0, err
	}
	return teamID, nil
}

func (r *PostgresUserRepository) GetTeamByUserID(userID string) (*domain.Team, error) {
	query := `
		SELECT t.id, t.name
		FROM teams t
		JOIN users u ON u.team_id = t.id
		WHERE u.id = $1
	`
	var team domain.Team
	err := r.db.QueryRow(query, userID).Scan(&team.ID, &team.Name)
	if err != nil {
		return nil, err
	}
	return &team, nil
}

func (r *PostgresUserRepository) DeactivateUsers(userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(userIDs))
	args := make([]interface{}, len(userIDs))
	for i, id := range userIDs {
		placeholders[i] = "$" + strconv.Itoa(i+1)
		args[i] = id
	}

	query := fmt.Sprintf("UPDATE users SET is_active = false WHERE id IN (%s)", strings.Join(placeholders, ","))
	_, err := r.db.Exec(query, args...)
	return err
}

func (r *PostgresUserRepository) SetIsActive(userID string, isActive bool) error {
	_, err := r.db.Exec("UPDATE users SET is_active = $1 WHERE id = $2", isActive, userID)
	return err
}

func (r *PostgresUserRepository) GetUserByID(userID string) (*domain.User, error) {
	query := `
		SELECT u.id, u.username, u.is_active, t.name, u.team_id
		FROM users u
		JOIN teams t ON u.team_id = t.id
		WHERE u.id = $1
	`
	var user domain.User
	var teamName string
	err := r.db.QueryRow(query, userID).Scan(
		&user.ID, &user.Username, &user.IsActive, &teamName, &user.TeamID,
	)
	if err != nil {
		return nil, err
	}
	user.TeamName = teamName
	return &user, nil
}
