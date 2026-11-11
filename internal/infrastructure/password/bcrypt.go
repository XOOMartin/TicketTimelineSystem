package password

import "golang.org/x/crypto/bcrypt"

// Bcrypt is a user.PasswordHasher.
type Bcrypt struct{}

func NewBcrypt() Bcrypt { return Bcrypt{} }

func (Bcrypt) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func (Bcrypt) Matches(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
