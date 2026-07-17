// Package apiclient provides helpers for making REST API calls,
// anything from handling the client, validating stuff,
// error handling, URL building, etc.
package apiclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/siggib007/goutils/logger"
)

// APIResponse holds the result of an API call
type APIResponse struct {
	Success bool
	Data    any
	Error   string
}

// APIClient handles HTTP API calls with rate limiting and proxy support
type APIClient struct {
	HTTPClient *http.Client
	Proxy      string
	MinQuiet   time.Duration
	TimeOut    time.Duration
	LastCall   time.Time
	StatusCode int
	TotalSleep time.Duration
	mu         sync.Mutex
	Logger     *logger.Logger
}

// NewAPIClient creates a new APIClient
func NewAPIClient(strProxy string, iTimeOut int, iMinQuiet int, objLogger *logger.Logger) *APIClient {
	objTransport := &http.Transport{
		TLSClientConfig: GetTLSConfig(),
	}
	if strProxy != "" {
		objProxyURL, err := url.Parse(strProxy)
		if err == nil {
			objTransport.Proxy = http.ProxyURL(objProxyURL)
		}
	}
	return &APIClient{
		HTTPClient: &http.Client{
			Timeout:   time.Duration(iTimeOut) * time.Second,
			Transport: objTransport,
		},
		Proxy:    strProxy,
		MinQuiet: time.Duration(iMinQuiet) * time.Second,
		TimeOut:  time.Duration(iTimeOut) * time.Second,
		Logger:   objLogger,
	}
}

func (a *APIClient) rateLimit() {
	a.mu.Lock()
	defer a.mu.Unlock()

	fDelta := time.Since(a.LastCall)
	a.Logger.LogEntry(fmt.Sprintf("It's been %.2f seconds since last API call", fDelta.Seconds()), 4, false)

	if fDelta < a.MinQuiet {
		iWait := a.MinQuiet - fDelta
		a.Logger.LogEntry(fmt.Sprintf("Waiting %.2f seconds before next API call", iWait.Seconds()), 4, false)
		a.TotalSleep += iWait
		time.Sleep(iWait)
	}
	a.LastCall = time.Now()
}

// APICallOptions holds all parameters for MakeAPICall.
//
// Body precedence when StrMethod is POST: LstFiles (multipart) takes
// priority over StrRawBody, which takes priority over DictPayload
// (JSON). Only one body-construction path runs per call.
//
// When StrRawBody is set, the caller is responsible for setting the
// correct Content-Type in DictHeader — MakeAPICall will not infer or
// overwrite it, unlike the JSON and multipart paths which set their
// own Content-Type automatically.
type APICallOptions struct {
	URL     string
	Header  map[string]string
	Method  string
	Payload any
	RawBody string
	Files   map[string]string
	UserID  string
	PWD     string
}

// MakeAPICall makes an HTTP API call per the given options.
func (a *APIClient) MakeAPICall(objOpts APICallOptions) APIResponse {
	a.rateLimit()

	a.Logger.LogEntry(fmt.Sprintf("Doing a %s to URL: %s", objOpts.Method, objOpts.URL), 1, false)

	var objBody io.Reader
	var strContentType string

	switch strings.ToLower(objOpts.Method) {
	case "get", "delete":
		objBody = nil

	case "post":
		if len(objOpts.Files) > 0 {
			objBuf := &bytes.Buffer{}
			objWriter := multipart.NewWriter(objBuf)

			if objOpts.Payload != nil {
				jsonBytes, err := json.Marshal(objOpts.Payload)
				if err != nil {
					return APIResponse{Success: false, Error: err.Error()}
				}
				objPart, err := objWriter.CreateFormField("data")
				if err != nil {
					return APIResponse{Success: false, Error: err.Error()}
				}
				if _, err := objPart.Write(jsonBytes); err != nil {
					return APIResponse{Success: false, Error: err.Error()}
				}
			}

			for strKey, strFilePath := range objOpts.Files {
				objFile, err := os.Open(strFilePath)
				if err != nil {
					return APIResponse{Success: false, Error: fmt.Sprintf("unable to open attachment %s: %s", strFilePath, err.Error())}
				}
				defer func() {
					_ = objFile.Close()
				}()
				objPart, err := objWriter.CreateFormFile(strKey, filepath.Base(strFilePath))
				if err != nil {
					return APIResponse{Success: false, Error: err.Error()}
				}
				if _, err := io.Copy(objPart, objFile); err != nil {
					return APIResponse{Success: false, Error: err.Error()}
				}
			}
			if err := objWriter.Close(); err != nil {
				return APIResponse{Success: false, Error: err.Error()}
			}
			objBody = objBuf
			strContentType = objWriter.FormDataContentType()

		} else if objOpts.RawBody != "" {
			objBody = strings.NewReader(objOpts.RawBody)
			// strContentType intentionally left unset — caller supplies it
			// via DictHeader (e.g. "application/x-www-form-urlencoded")

		} else if objOpts.Payload != nil {
			jsonBytes, err := json.Marshal(objOpts.Payload)
			if err != nil {
				return APIResponse{Success: false, Error: err.Error()}
			}
			objBody = bytes.NewReader(jsonBytes)
			strContentType = "application/json"
		}
	}

	objReq, err := http.NewRequest(strings.ToUpper(objOpts.Method), objOpts.URL, objBody)
	if err != nil {
		return APIResponse{Success: false, Error: err.Error()}
	}

	for strKey, strVal := range objOpts.Header {
		objReq.Header.Set(strKey, strVal)
	}
	if strContentType != "" {
		objReq.Header.Set("Content-Type", strContentType)
	}
	if objOpts.UserID != "" {
		objReq.SetBasicAuth(objOpts.UserID, objOpts.PWD)
	}

	// Scrub secrets from log
	dictLogHeader := make(map[string]string)
	for strKey, strVal := range objOpts.Header {
		if strings.ToLower(strKey) == "authorization" {
			dictLogHeader[strKey] = strVal[:10] + "**********"
		} else {
			dictLogHeader[strKey] = strVal
		}
	}
	a.Logger.LogEntry(fmt.Sprintf("Headers: %v", dictLogHeader), 4, false)

	objResp, err := a.HTTPClient.Do(objReq)
	if err != nil {
		return APIResponse{Success: false, Error: err.Error()}
	}
	defer func() {
		_ = objResp.Body.Close()
	}()

	a.StatusCode = objResp.StatusCode
	a.Logger.LogEntry(fmt.Sprintf("Call resulted in status code %d", a.StatusCode), 3, false)

	objRespBody, err := io.ReadAll(objResp.Body)
	if err != nil {
		return APIResponse{Success: false, Error: err.Error()}
	}
	strPreview := string(objRespBody)
	a.Logger.LogEntry(fmt.Sprintf("Response from API was:\n%v", strPreview), 8, false)
	if len(strPreview) > 100 {
		strPreview = strPreview[:100] + "..."
	}

	if objResp.StatusCode != 200 && objResp.StatusCode != 201 && objResp.StatusCode != 204 {
		a.Logger.LogEntry(fmt.Sprintf("HTTP Error: %d - %s", objResp.StatusCode, strPreview), 3, false)
		return APIResponse{Success: false, Error: fmt.Sprintf("HTTP %d: %s", objResp.StatusCode, strPreview)}
	}

	strRespText := string(objRespBody)
	if strings.HasPrefix(strRespText[:min(99, len(strRespText))], "<html>") || strRespText == "" {
		return APIResponse{Success: false, Error: "response was HTML or empty"}
	}

	if err := ValidateJSONShape(objRespBody); err != nil {
		return APIResponse{Success: false, Error: err.Error()}
	}

	var objResult any
	if err := json.Unmarshal(objRespBody, &objResult); err != nil {
		return APIResponse{Success: false, Error: fmt.Sprintf("failed to parse JSON: %s", err.Error())}
	}

	return APIResponse{Success: true, Data: objResult}
}

// BuildURL is a helper function for constructing a valid URL with parameters and all
// Detects if API simulator is being used sends back just the BaseURL
func BuildURL(strBaseURL string, strEndPoint string, dictParams map[string]string) string {
	if strings.Contains(strBaseURL, "apisim") {
		return strBaseURL
	}

	if !strings.HasSuffix(strBaseURL, "/") {
		strBaseURL += "/"
	}

	if len(dictParams) == 0 {
		return strBaseURL + strEndPoint
	}

	objValues := url.Values{}
	for strKey, strValue := range dictParams {
		objValues.Set(strKey, strValue)
	}
	strParams := objValues.Encode()

	return strBaseURL + strEndPoint + "?" + strParams
}

// ValidateJSONShape performs a cheap guard check that a response body
// looks like top-level JSON (object or array) before attempting to
// unmarshal it. It does not validate full JSON syntax — json.Unmarshal
// still does that — this only exists to produce a clear, specific error
// when a caller receives HTML, XML, or plaintext instead of JSON,
// rather than json.Unmarshal's generic parse error.
func ValidateJSONShape(bBody []byte) error {
	strTrimmed := strings.TrimSpace(string(bBody))
	if len(strTrimmed) == 0 {
		return errors.New("response body is empty")
	}

	cFirst := strTrimmed[0]
	if cFirst != '{' && cFirst != '[' {
		strPreview := strTrimmed
		if len(strPreview) > 100 {
			strPreview = strPreview[:100] + "..."
		}
		return fmt.Errorf("response does not look like JSON: %s", strPreview)
	}

	return nil
}
