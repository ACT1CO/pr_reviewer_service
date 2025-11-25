package domain

type Team struct {
	ID      int64
	Name    string
	Members []User
}
