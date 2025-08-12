package pkg

import (
	"bytes"
	"context"
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

	brevo "github.com/getbrevo/brevo-go/lib"
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
	length := 4
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

func setupEmail()  (*brevo.APIClient, error) {
	//email config setup
	brevoApiKey := os.Getenv("BREVO_KEY")

	var ctx context.Context
	cfg := brevo.NewConfiguration()
	cfg.AddDefaultHeader("api-key", brevoApiKey)
	cfg.AddDefaultHeader("partner-key", brevoApiKey)

	br := brevo.NewAPIClient(cfg)
	_, _, err := br.AccountApi.GetAccount(ctx)
	if err != nil {
		fmt.Println("Error when calling AccountApi->get_account: ", err.Error())
		return nil,err
	}
	return br,nil
}

// one for password resets, one for verfication
func SendVerficationEmail(email string, name string, verfier *models.EmailVerification) {
	type EmailData struct {
		Name    string
		Code    string
		Expires string
	}
	templ, err := template.ParseFiles("pkg/templates/VerifyAccount.html")
	if err != nil {
		panic(err)
	}
	var renderedHTML bytes.Buffer
	exipres_in := time.Until(verfier.ExpiresAt).Seconds()
	data := EmailData{
		Name:    name,
		Code:    verfier.Code,
		Expires: strconv.FormatFloat(exipres_in, 'f', -1, 64),
	}

	err = templ.Execute(&renderedHTML, data)

	if err != nil {
		fmt.Println(err.Error())
	}
	apiInstance, err := setupEmail()

	if err != nil {
		fmt.Println(err.Error())
	}

	details := brevo.SendSmtpEmail{
		Sender: &brevo.SendSmtpEmailSender{
			Name:  "CampusClaim",
			Email: "no-reply@campusclaim.tech",
		},
		To: []brevo.SendSmtpEmailTo{
			{Name: "CampusClaim", Email: email},
		},
		Subject:     "Verify your account",
		HtmlContent: renderedHTML.String(),
	}
	var ctx context.Context
	apiInstance.TransactionalEmailsApi.SendTransacEmail(ctx, details)
}

func SendResetEmail(email string, token string) {
	type EmailData struct {
		Url string
	}
	reset_url := os.Getenv("RESET_LINK")

	var renderedHTML bytes.Buffer
	data := EmailData{
		Url: fmt.Sprintf("%s?token=%s",reset_url, token),
	}
	templ, err := template.ParseFiles("pkg/templates/ResetPassword.html")
	
	if err != nil {
		fmt.Println(err)
	}

	err = templ.Execute(&renderedHTML, data)

	if err != nil {
		fmt.Println(err)
	}

	apiInstance, err := setupEmail()

	if err != nil {
		fmt.Println(err)
	}

	details := brevo.SendSmtpEmail{
		Sender: &brevo.SendSmtpEmailSender{
			Name:  "CampusClaim",
			Email: "no-reply@campusclaim.tech",
		},
		To: []brevo.SendSmtpEmailTo{
			{Name: "CampusClaim", Email: email},
		},
		Subject:     "Change Account Password",
		HtmlContent: renderedHTML.String(),
	}
	var ctx context.Context
	_, _, err = apiInstance.TransactionalEmailsApi.SendTransacEmail(ctx, details)
	if (err != nil) {
		fmt.Println(err)
	}
	fmt.Println("Reset Email sent")
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
func SendAddImageURL(imageURL, text, requestType, item_uuid string) (map[string]interface{}, error) {
	var endpoint string
	requestBody := make(map[string]string)

	if requestType == "search" {
		endpoint = os.Getenv("SEARCH_ENDPOINT")
		requestBody["image_url"] = imageURL
		requestBody["text"] = text
	} else { // Default to "add"
		endpoint = os.Getenv("ADD_ENDPOINT")
		requestBody["image_url"] = imageURL
		requestBody["item_id"] = item_uuid
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
