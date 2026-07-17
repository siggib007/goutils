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
	BSuccess bool
	ObjData  any
	StrError string
}

// APIClient handles HTTP API calls with rate limiting and proxy support
type APIClient struct {
	objHTTP     *http.Client
	strProxy    string
	iMinQuiet   time.Duration
	iTimeOut    time.Duration
	tLastCall   time.Time
	IStatusCode int
	iTotalSleep time.Duration
	mu          sync.Mutex
	objLogger   *logger.Logger
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
		objHTTP: &http.Client{
			Timeout:   time.Duration(iTimeOut) * time.Second,
			Transport: objTransport,
		},
		iMinQuiet: time.Duration(iMinQuiet) * time.Second,
		iTimeOut:  time.Duration(iTimeOut) * time.Second,
		objLogger: objLogger,
	}
}

func (a *APIClient) rateLimit() {
	a.mu.Lock()
	defer a.mu.Unlock()

	fDelta := time.Since(a.tLastCall)
	a.objLogger.LogEntry(fmt.Sprintf("It's been %.2f seconds since last API call", fDelta.Seconds()), 4, false)

	if fDelta < a.iMinQuiet {
		iWait := a.iMinQuiet - fDelta
		a.objLogger.LogEntry(fmt.Sprintf("Waiting %.2f seconds before next API call", iWait.Seconds()), 4, false)
		a.iTotalSleep += iWait
		time.Sleep(iWait)
	}
	a.tLastCall = time.Now()
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
	StrURL      string
	DictHeader  map[string]string
	StrMethod   string
	DictPayload any
	StrRawBody  string
	LstFiles    map[string]string
	StrUser     string
	StrPWD      string
}

// MakeAPICall makes an HTTP API call per the given options.
func (a *APIClient) MakeAPICall(objOpts APICallOptions) APIResponse {
	a.rateLimit()

	a.objLogger.LogEntry(fmt.Sprintf("Doing a %s to URL: %s", objOpts.StrMethod, objOpts.StrURL), 1, false)

	var objBody io.Reader
	var strContentType string

	switch strings.ToLower(objOpts.StrMethod) {
	case "get", "delete":
		objBody = nil

	case "post":
		if len(objOpts.LstFiles) > 0 {
			objBuf := &bytes.Buffer{}
			objWriter := multipart.NewWriter(objBuf)

			if objOpts.DictPayload != nil {
				jsonBytes, err := json.Marshal(objOpts.DictPayload)
				if err != nil {
					return APIResponse{BSuccess: false, StrError: err.Error()}
				}
				objPart, err := objWriter.CreateFormField("data")
				if err != nil {
					return APIResponse{BSuccess: false, StrError: err.Error()}
				}
				objPart.Write(jsonBytes)
			}

			for strKey, strFilePath := range objOpts.LstFiles {
				objFile, err := os.Open(strFilePath)
				if err != nil {
					return APIResponse{BSuccess: false, StrError: fmt.Sprintf("unable to open attachment %s: %s", strFilePath, err.Error())}
				}
				defer objFile.Close()
				objPart, err := objWriter.CreateFormFile(strKey, filepath.Base(strFilePath))
				if err != nil {
					return APIResponse{BSuccess: false, StrError: err.Error()}
				}
				io.Copy(objPart, objFile)
			}
			objWriter.Close()
			objBody = objBuf
			strContentType = objWriter.FormDataContentType()

		} else if objOpts.StrRawBody != "" {
			objBody = strings.NewReader(objOpts.StrRawBody)
			// strContentType intentionally left unset — caller supplies it
			// via DictHeader (e.g. "application/x-www-form-urlencoded")

		} else if objOpts.DictPayload != nil {
			jsonBytes, err := json.Marshal(objOpts.DictPayload)
			if err != nil {
				return APIResponse{BSuccess: false, StrError: err.Error()}
			}
			objBody = bytes.NewReader(jsonBytes)
			strContentType = "application/json"
		}
	}

	objReq, err := http.NewRequest(strings.ToUpper(objOpts.StrMethod), objOpts.StrURL, objBody)
	if err != nil {
		return APIResponse{BSuccess: false, StrError: err.Error()}
	}

	for strKey, strVal := range objOpts.DictHeader {
		objReq.Header.Set(strKey, strVal)
	}
	if strContentType != "" {
		objReq.Header.Set("Content-Type", strContentType)
	}
	if objOpts.StrUser != "" {
		objReq.SetBasicAuth(objOpts.StrUser, objOpts.StrPWD)
	}

	// Scrub secrets from log
	dictLogHeader := make(map[string]string)
	for strKey, strVal := range objOpts.DictHeader {
		if strings.ToLower(strKey) == "authorization" {
			dictLogHeader[strKey] = strVal[:10] + "**********"
		} else {
			dictLogHeader[strKey] = strVal
		}
	}
	a.objLogger.LogEntry(fmt.Sprintf("Headers: %v", dictLogHeader), 4, false)

	objResp, err := a.objHTTP.Do(objReq)
	if err != nil {
		return APIResponse{BSuccess: false, StrError: err.Error()}
	}
	defer objResp.Body.Close()

	a.IStatusCode = objResp.StatusCode
	a.objLogger.LogEntry(fmt.Sprintf("Call resulted in status code %d", a.IStatusCode), 3, false)

	objRespBody, err := io.ReadAll(objResp.Body)
	if err != nil {
		return APIResponse{BSuccess: false, StrError: err.Error()}
	}
	strPreview := string(objRespBody)
	a.objLogger.LogEntry(fmt.Sprintf("Response from API was:\n%v", strPreview), 8, false)
	if len(strPreview) > 100 {
		strPreview = strPreview[:100] + "..."
	}

	if objResp.StatusCode != 200 && objResp.StatusCode != 201 && objResp.StatusCode != 204 {
		a.objLogger.LogEntry(fmt.Sprintf("HTTP Error: %d - %s", objResp.StatusCode, strPreview), 3, false)
		return APIResponse{BSuccess: false, StrError: fmt.Sprintf("HTTP %d: %s", objResp.StatusCode, strPreview)}
	}

	strRespText := string(objRespBody)
	if strings.HasPrefix(strRespText[:min(99, len(strRespText))], "<html>") || strRespText == "" {
		return APIResponse{BSuccess: false, StrError: "response was HTML or empty"}
	}

	if err := ValidateJSONShape(objRespBody); err != nil {
		return APIResponse{BSuccess: false, StrError: err.Error()}
	}

	var objResult any
	if err := json.Unmarshal(objRespBody, &objResult); err != nil {
		return APIResponse{BSuccess: false, StrError: fmt.Sprintf("failed to parse JSON: %s", err.Error())}
	}

	return APIResponse{BSuccess: true, ObjData: objResult}
}

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
