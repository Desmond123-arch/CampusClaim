package pkg

import "golang.org/x/crypto/bcrypt"


func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func VerifyHash(password string, hashstring string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashstring), []byte(password))
	return err == nil
}