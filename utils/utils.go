package utils

import (
	"bufio"
	"errors"
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
func GetInput(strPrompt string) string {
	fmt.Print(strPrompt)
	var strInput string
	fmt.Scanln(&strInput)
	return strings.TrimSpace(strInput)
}

// ChkDir checks if a directory exists and creates it if not
// Returns true if directory exists or was created successfully
func ChkDir(strDir string) bool {
	if _, err := os.Stat(strDir); os.IsNotExist(err) {
		if err := os.MkdirAll(strDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Unable to create directory %s: %s\n", strDir, err)
			return false
		}
	}
	return true
}

// ListFiles returns files in a directory matching a pattern
func ListFiles(strDirectory string, strPattern string) []string {
	var lstFiles []string
	objEntries, err := os.ReadDir(strDirectory)
	if err != nil {
		return lstFiles
	}
	for _, objEntry := range objEntries {
		if !objEntry.IsDir() {
			if MatchPattern(objEntry.Name(), strPattern) {
				lstFiles = append(lstFiles, objEntry.Name())
			}
		}
	}
	return lstFiles
}

func FindFilesExt(strDirectory string, strExt string) []string {
	var lstFiles []string
	objEntries, err := os.ReadDir(strDirectory)
	if err != nil {
		return lstFiles
	}
	for _, objEntry := range objEntries {
		if !objEntry.IsDir() {
			if strings.EqualFold(filepath.Ext(objEntry.Name()), strExt) {
				lstFiles = append(lstFiles, objEntry.Name())
			}
		}
	}
	return lstFiles
}

// CheckPath reports whether strPath exists, whether it is a directory,
// and whether it was given as a fully qualified (absolute) path.
func CheckPath(strPath string) (bIsDir bool, bIsAbsolute bool, err error) {
	if strPath == "" {
		return false, false, fmt.Errorf("CheckPath: empty path provided")
	}

	bIsAbsolute = filepath.IsAbs(strPath)

	strAbsPath, errAbs := filepath.Abs(strPath)
	if errAbs != nil {
		strAbsPath = strPath // fall back to original, still attempt Stat
	}

	objInfo, err := os.Stat(strAbsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, bIsAbsolute, fmt.Errorf("CheckPath: path does not exist: %s", strAbsPath)
		}
		return false, bIsAbsolute, fmt.Errorf("CheckPath: unable to stat path %s: %w", strAbsPath, err)
	}

	bIsDir = objInfo.IsDir()
	return bIsDir, bIsAbsolute, nil
}

func FileExists(strPath string) bool {
	objInfo, err := os.Stat(strPath)
	if err != nil {
		return false
	}
	return !objInfo.IsDir()
}

func DirExists(strPath string) bool {
	objInfo, err := os.Stat(strPath)
	if err != nil {
		return false
	}
	return objInfo.IsDir()
}

// PathConfig holds the base paths resolved at startup.
type PathConfig struct {
	DefConf    string // default config file path
	DefLogFile string // default log file path
	AppName    string // name of the running exe
	ExeDir     string // directory containing the executable
}

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

	strLine := objScanner.Text()
	return strLine, nil
}
