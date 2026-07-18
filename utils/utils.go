// Package utils contains a collection of handy utilities
package utils

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// IsInt returns true if the string can be parsed as a non-negative integer
func IsInt(strVal string) bool {
	if strVal == "" {
		return false
	}
	for _, c := range strVal {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// ParseFloat converts a string to float64, returns 0.0 on failure
func ParseFloat(strVal string) float64 {
	fVal, err := strconv.ParseFloat(strings.TrimSpace(strVal), 64)
	if err != nil {
		return 0.0
	}
	return fVal
}

// MatchPattern checks if strName matches a simple wildcard pattern
// Only supports trailing * wildcard e.g. "12345*"
func MatchPattern(strName string, strPattern string) bool {
	if !strings.Contains(strPattern, "*") {
		return strName == strPattern
	}
	strPrefix := strPattern[:strings.Index(strPattern, "*")]
	return strings.HasPrefix(strName, strPrefix)
}

// GetInput prints a prompt and reads a line from stdin
// Not designed for non-interactive situations like piping stdin.
func GetInput(strPrompt string) string {
	fmt.Print(strPrompt)
	var strInput string
	_, objErr := fmt.Scanln(&strInput)
	if objErr != nil {
		fmt.Printf("Issue reading console: %v\n", objErr)
	}
	strLine := strings.TrimSpace(strInput)
	return StripQuotes(strLine)
}

// PathConfig holds the base paths resolved at startup.
type PathConfig struct {
	DefConf    string // default config file path
	DefLogFile string // default log file path
	AppName    string // name of the running exe
	ExeDir     string // directory containing the executable
}

// BasePaths is a helperfunction to calculate execution path,
// appName, default log and default config file both fully qualified
func BasePaths() (*PathConfig, error) {
	// Establish base directory and script name
	strExePath, err := os.Executable()
	if err != nil {
		return nil, err
	}
	strExeDir := filepath.Dir(strExePath)
	strAppName := filepath.Base(strExePath)

	strISO := time.Now().Format("-2006-01-02T15-04-05")

	// Log directory
	strLogDir := filepath.Join(strExeDir, "Logs")
	if !ChkDir(strLogDir) {
		return nil, err
	}

	// Default config and log file paths
	strBaseName := strAppName
	iDotPos := strings.LastIndex(strAppName, ".")
	if iDotPos >= 1 {
		strBaseName = strAppName[:iDotPos]
	}
	strConfName := "config.ini"
	strDefConf := filepath.Join(strExeDir, strConfName)

	strLogFilename := strBaseName + strISO + ".log"
	strDefLogFile := filepath.Join(strLogDir, strLogFilename)

	objPaths := &PathConfig{
		DefConf:    strDefConf,
		DefLogFile: strDefLogFile,
		AppName:    strAppName,
		ExeDir:     strExeDir,
	}

	return objPaths, nil
}

// ReadLine prompts on stdout (if strPrompt is non-empty) and reads a
// single line from stdin, spaces and all. Returns an error if stdin
// is closed/exhausted before a line is read, or if the scanner itself
// fails (e.g. an underlying I/O error).
// Not designed for non-interactive situations like piping stdin.
func ReadLine(strPrompt string) (string, error) {
	if strPrompt != "" {
		fmt.Print(strPrompt)
	}

	objScanner := bufio.NewScanner(os.Stdin)

	bHasLine := objScanner.Scan()
	if !bHasLine {
		objErr := objScanner.Err()
		if objErr != nil {
			return "", fmt.Errorf("failed to read line: %w", objErr)
		}
		return "", fmt.Errorf("no input received (stdin closed)")
	}

	strLine := strings.TrimSpace(objScanner.Text())
	strLine = StripQuotes(strLine)
	return strLine, nil
}

// StripQuotes removes a single matching pair of leading/trailing quote
// characters (" or ') if present. Unquoted or mismatched-quote strings
// are returned unchanged.
func StripQuotes(strInput string) string {
	if len(strInput) < 2 {
		return strInput
	}

	chFirst := strInput[0]
	chLast := strInput[len(strInput)-1]
	bIsQuoteChar := chFirst == '"' || chFirst == '\''

	if bIsQuoteChar && chFirst == chLast {
		return strInput[1 : len(strInput)-1]
	}

	return strInput
}
