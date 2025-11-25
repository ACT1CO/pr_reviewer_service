package domain

type User struct {
	ID       string
	Username string
	IsActive bool
	TeamID   int64
	TeamName string
}
