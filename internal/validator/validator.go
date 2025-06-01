package validator

import (
	"net/url"
	"strings"

	"github.com/go-playground/validator/v10"
)

type URLValidator struct {
	validate *validator.Validate
}

func NewURLValidator() *URLValidator {
	v := validator.New()
	v.RegisterValidation("http_protocol", validateHTTPProtocol) //nolint:errcheck
	return &URLValidator{
		validate: v,
	}
}

func validateHTTPProtocol(fl validator.FieldLevel) bool {
	urlStr := fl.Field().String()
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	return strings.HasPrefix(parsedURL.Scheme, "http")
}

func (v *URLValidator) ValidateURL(url string) error {
	type urlStruct struct {
		URL string `validate:"required,url,http_protocol"`
	}

	return v.validate.Struct(urlStruct{URL: url})
}

func (v *URLValidator) ValidateURLs(urls []string) ValidationResults {
	results := make([]ValidationResult, len(urls))

	for i, url := range urls {
		results[i] = ValidationResult{
			URL:   url,
			Index: i + 1,
			Error: v.ValidateURL(url),
		}
	}

	return results
}

type ValidationResult struct {
	URL   string
	Index int
	Error error
}

func (r ValidationResult) IsValid() bool {
	return r.Error == nil
}

type ValidationResults []ValidationResult

func (vr ValidationResults) GetInvalidURLs() []string {
	invalidURLs := make([]string, 0)
	for _, result := range vr {
		if !result.IsValid() {
			invalidURLs = append(invalidURLs, result.URL)
		}
	}
	return invalidURLs
}

func HasInvalidURLs(results ValidationResults) bool {
	for _, result := range results {
		if !result.IsValid() {
			return true
		}
	}
	return false
}
