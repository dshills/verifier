package testdata

import (
	"database/sql"
	"net/http"
)

// CreateUser is a function that returns an error.
func CreateUser(name string) error {
	return nil
}

// HandleHealth is an HTTP handler.
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// QueryUsers has a DB call.
func QueryUsers(db *sql.DB) error {
	_, err := db.Query("SELECT * FROM users")
	return err
}

// ComplexFunc has high branching.
func ComplexFunc(x int) string {
	switch {
	case x < 0:
		return "negative"
	case x == 0:
		return "zero"
	case x == 1:
		return "one"
	case x == 2:
		return "two"
	case x == 3:
		return "three"
	case x > 100:
		return "large"
	default:
		return "other"
	}
}

// ValidateInput checks input.
func ValidateInput(s string) bool {
	if s == "" {
		return false
	}
	return len(s) < 100
}

// MyInterface is an exported interface.
type MyInterface interface {
	DoSomething() error
}

// UserService is a type.
type UserService struct {
	db *sql.DB
}

// GetUser is a method on UserService.
func (s *UserService) GetUser(id int) error {
	return nil
}
