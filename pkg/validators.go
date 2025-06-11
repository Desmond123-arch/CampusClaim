package pkg

import (
	// "fmt"

	"fmt"
	"regexp"

	"github.com/go-playground/validator/v10"
)

type (
	ErrorResponse struct {
		// Error string
		FailedField string
		Tag         string
		Value       interface{}
		Message     string
	}
	XValidator struct {
		validator *validator.Validate
	}

	GlobalErrorHandlerResp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
)

// var validate = validator.New()

func (v XValidator) Validate(data interface{}) []ErrorResponse {
	validationErrors := []ErrorResponse{}
	errs := v.validator.Struct(data)

	if errs != nil {
		for _, err := range errs.(validator.ValidationErrors) {
			msg := "Validation failed on field " + err.Field()

			switch err.Tag() {
			case "required":
				msg = fmt.Sprintf("%s is required", err.Field())
			case "school_email":
				msg = fmt.Sprintf("%s must be a valid UMAT student email", err.Field())
			case "validate_password":
				msg = "Confirm Password must match with Password"
			}

			var elem ErrorResponse

			elem.FailedField = err.Field()
			elem.Value = err.Value()
			// elem.Error = err.Error()
			elem.Message = msg
			elem.Tag = err.Tag()
			validationErrors = append(validationErrors, elem)
		}
	}
	return validationErrors
}

func isSchoolEmail(f1 validator.FieldLevel) bool {
	email := f1.Field().String()
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9._%+-]+@st\.umat\.edu\.gh$`, email)
	return matched
}
func validatePassword(f1 validator.FieldLevel) bool {
	parent := f1.Parent()
	confirmField := parent.FieldByName("ConfirmPassword")

	if !confirmField.IsValid() {
		return false
	}

	confirmPassword := confirmField.String()
	password := f1.Field().String()

	return confirmPassword == password
}

func RegistrationValidatator() *XValidator {
	v := validator.New()
	_ = v.RegisterValidation("school_email", isSchoolEmail)
	_ = v.RegisterValidation("validate_password", validatePassword)
	return &XValidator{validator: v}
}

func LoginValidator() *XValidator {
	v := validator.New()
	_ = v.RegisterValidation("school_email", isSchoolEmail)
	return &XValidator{validator: v}

}

func GeneralValidator() *XValidator {
	v := validator.New()
	return &XValidator{validator: v}
}