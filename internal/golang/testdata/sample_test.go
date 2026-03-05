package testdata

import "testing"

func TestCreateUser(t *testing.T) {
	t.Run("valid name", func(t *testing.T) {
		if err := CreateUser("alice"); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("empty name", func(t *testing.T) {
		_ = CreateUser("")
	})
}

func TestValidateInput(t *testing.T) {
	if ValidateInput("") {
		t.Error("expected false for empty")
	}
}
