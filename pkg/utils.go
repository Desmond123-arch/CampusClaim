package pkg

import (
	"crypto/rand"

	"golang.org/x/crypto/bcrypt"
)


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

const otpChars = "1234567890"

func GenerateOTP() (string, error) {
	length := 6
    buffer := make([]byte, length)
    _, err := rand.Read(buffer)
    if err != nil {
        return "", err
    }

    otpCharsLength := len(otpChars)
    for i := range length {
        buffer[i] = otpChars[int(buffer[i])%otpCharsLength]
    }

    return string(buffer), nil
}