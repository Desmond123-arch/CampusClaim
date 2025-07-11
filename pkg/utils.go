package pkg

import (
	"bytes"
	// "context"
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


// Client is a client for interacting with the image search API.
type Client struct {
	addEndpoint    string
	searchEndpoint string
	httpClient     *http.Client
}

// NewClient creates a new API client.
func NewClient() (*Client, error) {
	addEndpoint := os.Getenv("ADD_ENDPOINT")
	if addEndpoint == "" {
		return nil, fmt.Errorf("ADD_ENDPOINT environment variable not set")
	}

	searchEndpoint := os.Getenv("SEARCH_ENDPOINT")
	if searchEndpoint == "" {
		return nil, fmt.Errorf("SEARCH_ENDPOINT environment variable not set")
	}

	return &Client{
		addEndpoint:    addEndpoint,
		searchEndpoint: searchEndpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // Set a reasonable timeout
		},
	}, nil
}


// It's still highly recommended to have a shared HTTP client with a timeout.
// This prevents your function from hanging indefinitely on a network issue.
// We can define it once at the package level.
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// Renamed for clarity, since it handles both add and search.
func SendAddImageURL(imageURL, text, requestType string) (map[string]interface{}, error) {
	var endpoint string
	requestBody := make(map[string]string)

	if requestType == "search" {
		endpoint = os.Getenv("SEARCH_ENDPOINT")
		requestBody["image_url"] = imageURL
		requestBody["text"] = text
	} else { // Default to "add"
		endpoint = os.Getenv("ADD_ENDPOINT")
		requestBody["image_url"] = imageURL  
		requestBody["description"] = text
	}
	
	if endpoint == "" {
		return nil, fmt.Errorf("%s environment variable not set", requestType)
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request JSON: %w", err)
	}

	resp, err := httpClient.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned an error: %s", resp.Status)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}

	return result, nil
}

