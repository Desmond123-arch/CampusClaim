package pkg

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"text/template"
	"time"

	"github.com/Desmond123-arch/CampusClaim/models"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/gomail.v2"
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

// one for password resets, one for verfication
func SendVerficationEmail(email string, name string, verfier *models.EmailVerification) {
	type EmailData struct {
		Name    string
		Code    string
		Expires string
	}
	password := os.Getenv("GMAIL_PASSWORD")
	m := gomail.NewMessage()
	m.SetHeader("From", "campusclaimumat@gmail.com")
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Verify your account")
	templ, err := template.ParseFiles("pkg/templates/VerifyAccount.html")
	if err != nil {
		panic(err)
	}
	var renderedHTML bytes.Buffer
	exipres_in := verfier.ExpiresAt.Sub(time.Now()).Seconds()
	data := EmailData{
		Name:    name,
		Code:    verfier.Code,
		Expires: strconv.FormatFloat(exipres_in, 'f', -1, 64),
	}

	err = templ.Execute(&renderedHTML, data)

	if err != nil {
		fmt.Println(err.Error())
	}

	m.SetBody("text/html", renderedHTML.String())

	d := gomail.NewDialer("smtp.gmail.com", 465, "campusclaimumat@gmail.com", password)

	if err := d.DialAndSend(m); err != nil {
		panic(err)
	}

	fmt.Println("Email sent")
}

func SendResetEmail(email string, token string) {
	type EmailData struct {
		Url string
	}
	password := os.Getenv("GMAIL_PASSWORD")
	m := gomail.NewMessage()
	m.SetHeader("From", "campusclaimumat@gmail.com")
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Reset Password")
	templ, err := template.ParseFiles("pkg/templates/ResetPassword.html")
	if err != nil {
		panic(err)
	}
	var renderedHTML bytes.Buffer
	data := EmailData{
		Url: fmt.Sprintf("https://campusclaim.com/reset-password?token=%s", token),
	}

	err = templ.Execute(&renderedHTML, data)

	if err != nil {
		fmt.Println(err.Error())
	}

	m.SetBody("text/html", renderedHTML.String())

	d := gomail.NewDialer("smtp.gmail.com", 465, "campusclaimumat@gmail.com", password)

	if err := d.DialAndSend(m); err != nil {
		panic(err)
	}

	fmt.Println("Email sent")
}

func SendAddImageURL(url, description, requestType string) (map[string]interface{}, error) {
	var endpoint string
	if requestType == "search" {
		endpoint = os.Getenv("SEARCH_ENDPOINT")
	} else {
		endpoint = os.Getenv("ADD_ENDPOINT")
	}
	requestBody := map[string]string{
		"image_url": endpoint,
		"text":      description,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	return result, nil
}
