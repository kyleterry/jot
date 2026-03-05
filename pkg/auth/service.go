package auth

// PasswordManagerService generates passwords for keys and can
// report if a supplied password is the correct one for a key.
type PasswordManagerService interface {
	// Generate generates a password for key
	Generate(key string) (string, error)
	// IsMatch reports if a supplied password can be created
	// with key
	IsMatch(key, supplied string) (bool, error)
}
