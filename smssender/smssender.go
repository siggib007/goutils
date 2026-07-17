package smssender

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

type SendOptions struct {
	MsgTo   string
	Message string
	AppName string
}

func SendSMS(objSendOption SendOptions, objCfg TwilioConfig, objLogger *logger.Logger) error {
	// Validate required config
	if objCfg.BaseURL == "" || objCfg.ClientID == "" || objCfg.ClientSecret == "" {
		return fmt.Errorf("SendSMS: Twilio credentials or URL missing")
	}

	if err := ValidateAlphanumericSenderId(objCfg.MsgFrom); err != nil {
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
	objCallOptions.StrURL = strURL
	objCallOptions.DictHeader = dictHeader
	objCallOptions.StrMethod = "POST"
	objCallOptions.StrRawBody = strEncoded
	objCallOptions.StrUser = objCfg.ClientID
	objCallOptions.StrPWD = objCfg.ClientSecret

	objLogger.Log("Posting Message")
	objResp := objAPI.MakeAPICall(objCallOptions)
	if !objResp.BSuccess {
		return fmt.Errorf("SendSMS: Failed to send message: %w", err)
	}
	dictResp, ok := objResp.ObjData.(map[string]any)
	if !ok {
		return errors.New("Unexpected response format")
	}
	strStatus, ok := dictResp["status"].(string)
	if !ok {
		return errors.New("No status in response")
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

const iMaxSenderIdLen = 11

var reValidSenderIdChars = regexp.MustCompile(`^[A-Za-z0-9 &_-]+$`)

// ValidateAlphanumericSenderId enforces Twilio's alphanumeric sender ID
// requirements: up to 11 characters, letters/digits/spaces plus
// hyphen, underscore, and ampersand only. Returns an error describing
// the specific violation rather than a bare pass/fail.
func ValidateAlphanumericSenderId(strSenderId string) error {
	if strSenderId == "" {
		return fmt.Errorf("sender ID is empty")
	}

	iLen := len(strSenderId)
	if iLen > iMaxSenderIdLen {
		return fmt.Errorf("sender ID %q is %d characters, exceeds max of %d", strSenderId, iLen, iMaxSenderIdLen)
	}

	if !reValidSenderIdChars.MatchString(strSenderId) {
		return fmt.Errorf("sender ID %q contains characters outside the allowed set (letters, digits, space, -, _, &)", strSenderId)
	}

	return nil
}
