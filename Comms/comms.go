// Package comms provides various communications helpers such as sms, slack and email
// For example SendSMS validates and sends SMS
// messages through the Twilio API, including sender ID validation and
// message body sanitization.
package comms

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/siggib007/goutils/apiclient"
	"github.com/siggib007/goutils/logger"
)

// TwilioConfig holds all the details needed for API call to Twilio
type TwilioConfig struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	MsgFrom      string
	Proxy        string
	MinQuiet     int
	TimeOut      int
	MaxMsgLen    int
}

// SendOptions holds all the message specific details
type SendOptions struct {
	MsgTo   string
	Message string
	AppName string
}

// SendSMS handles making the API call to the Twilio API that actually sends the message
func SendSMS(objSendOption SendOptions, objCfg TwilioConfig, objLogger *logger.Logger) error {
	// Validate required config
	if objCfg.BaseURL == "" || objCfg.ClientID == "" || objCfg.ClientSecret == "" {
		return fmt.Errorf("SendSMS: Twilio credentials or URL missing")
	}

	if err := ValidateAlphanumericSenderID(objCfg.MsgFrom); err != nil {
		return fmt.Errorf("SendSMS: invalid sender id: %w", err)
	}

	strPhone, err := SanitizePhone(objSendOption.MsgTo)
	if err != nil {
		return fmt.Errorf("SendSMS: invalid to phone number: %w", err)
	}

	strMsg, err := SanitizeSmsBody(objSendOption.Message, objCfg.MaxMsgLen)
	if err != nil {
		return fmt.Errorf("SendSMS: bad message: %w", err)
	}

	objValues := url.Values{}
	objValues.Set("From", objCfg.MsgFrom)
	objValues.Set("Body", strMsg)
	objValues.Set("To", strPhone)
	strEncoded := objValues.Encode()

	objAPI := apiclient.NewAPIClient(objCfg.Proxy, objCfg.TimeOut, objCfg.MinQuiet, objLogger)
	dictHeader := make(map[string]string)
	dictHeader["Content-Type"] = "application/x-www-form-urlencoded"
	dictHeader["Accept"] = "*/*"
	dictHeader["Application"] = objSendOption.AppName
	dictHeader["User-Agent"] = fmt.Sprintf("Go/%s", objSendOption.AppName)
	dictMyParams := make(map[string]string)
	strURL := apiclient.BuildURL(objCfg.BaseURL, objCfg.ClientID+"/Messages.json", dictMyParams)
	objCallOptions := apiclient.APICallOptions{}
	objCallOptions.URL = strURL
	objCallOptions.Header = dictHeader
	objCallOptions.Method = "POST"
	objCallOptions.RawBody = strEncoded
	objCallOptions.UserID = objCfg.ClientID
	objCallOptions.PWD = objCfg.ClientSecret

	objLogger.Log("Posting Message")
	objResp := objAPI.MakeAPICall(objCallOptions)
	if !objResp.Success {
		return fmt.Errorf("SendSMS: Failed to send message: %w", err)
	}
	dictResp, ok := objResp.Data.(map[string]any)
	if !ok {
		return errors.New("nexpected response format")
	}
	strStatus, ok := dictResp["status"].(string)
	if !ok {
		return errors.New("no status in response")
	}
	objLogger.Log(fmt.Sprintf("Status: %v", strStatus))
	return nil
}

var reNonDigit = regexp.MustCompile(`[^0-9]`)

// SanitizePhone strips formatting characters and validates that what's
// left looks like a plausible phone number. Returns an error (not nil)
// on any failure, so callers can fail loud instead of silently proceeding.
func SanitizePhone(strInput string) (string, error) {
	strTrimmed := strings.TrimSpace(strInput)
	if strTrimmed == "" {
		return "", fmt.Errorf("phone number is empty")
	}

	bHasLeadingPlus := strings.HasPrefix(strTrimmed, "+")

	strDigitsOnly := reNonDigit.ReplaceAllString(strTrimmed, "")
	if strDigitsOnly == "" {
		return "", fmt.Errorf("phone number %q contains no digits", strInput)
	}

	iLen := len(strDigitsOnly)
	if iLen < 7 || iLen > 15 {
		return "", fmt.Errorf("phone number %q has %d digits, expected 7-15", strInput, iLen)
	}

	strResult := strDigitsOnly
	if bHasLeadingPlus {
		strResult = "+" + strDigitsOnly
	}

	return strResult, nil
}

// SanitizeSmsBody removes control characters that have no legitimate
// place in message text, while leaving normal language, punctuation,
// and unicode untouched. Returns an error if the message is empty,
// oversized, or made entirely of characters that got stripped.
func SanitizeSmsBody(strInput string, iMaxMessageLen int) (string, error) {
	if strInput == "" {
		return "", fmt.Errorf("message body is empty")
	}

	strCleaned := strings.Map(func(rChar rune) rune {
		if rChar == '\n' || rChar == '\t' {
			return rChar
		}
		if unicode.IsControl(rChar) {
			return -1
		}
		return rChar
	}, strInput)

	strTrimmed := strings.TrimSpace(strCleaned)
	if strTrimmed == "" {
		return "", fmt.Errorf("message body contained no usable text after sanitization")
	}

	iLen := len([]rune(strTrimmed))
	if iLen > iMaxMessageLen {
		return "", fmt.Errorf("message body is %d characters, exceeds max of %d", iLen, iMaxMessageLen)
	}

	return strTrimmed, nil
}

const iMaxSenderIDLen = 11

var reValidSenderIDChars = regexp.MustCompile(`^[A-Za-z0-9 &_-]+$`)

// ValidateAlphanumericSenderID enforces Twilio's alphanumeric sender ID
// requirements: up to 11 characters, letters/digits/spaces plus
// hyphen, underscore, and ampersand only. Returns an error describing
// the specific violation rather than a bare pass/fail.
func ValidateAlphanumericSenderID(strSenderID string) error {
	if strSenderID == "" {
		return fmt.Errorf("sender ID is empty")
	}

	iLen := len(strSenderID)
	if iLen > iMaxSenderIDLen {
		return fmt.Errorf("sender ID %q is %d characters, exceeds max of %d", strSenderID, iLen, iMaxSenderIDLen)
	}

	if !reValidSenderIDChars.MatchString(strSenderID) {
		return fmt.Errorf("sender ID %q contains characters outside the allowed set (letters, digits, space, -, _, &)", strSenderID)
	}

	return nil
}
