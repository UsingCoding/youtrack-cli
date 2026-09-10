package domain

type User struct {
	ID       string
	Login    string
	FullName string
	Email    string
}

type Group struct {
	ID   string
	Name string
}
