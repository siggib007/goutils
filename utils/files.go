package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/siggib007/goutils/logger"
)

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

// FindFilesExt is a helper function to build a list of files of particular extension
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

// FileExists a small helper function to help determine if the path provided is valid,
// and if it is a file or directory. Returns false even if it is a valid path, but to a directory
func FileExists(strPath string) bool {
	objInfo, err := os.Stat(strPath)
	if err != nil {
		return false
	}
	return !objInfo.IsDir()
}

// DirExists a small helper function to help determine if the path provided is valid,
// and if it is a file or directory. Returns false even if it is a valid path, but to a file
func DirExists(strPath string) bool {
	objInfo, err := os.Stat(strPath)
	if err != nil {
		return false
	}
	return objInfo.IsDir()
}

// ValidateConfPath is a helper function that validates the path provided, and tries to find alternatives
func ValidateConfPath(objLogger *logger.Logger, strConfFile *string, bUseEnv bool, objPaths PathConfig) {
	if !bUseEnv {
		objLogger.Log(fmt.Sprintf("Config file set to: %s", *strConfFile))
		bFail := false
		bIsDir, _, err := CheckPath(*strConfFile)
		if err != nil {
			objLogger.LogEntry(fmt.Sprintf("Invalid config path: %v", err), 0, false)
			bFail = true
		}
		if bIsDir {
			objLogger.LogEntry("Config path, is just a directory not a file:", 0, false)
			bFail = true
		}
		if bFail {
			objLogger.Log(fmt.Sprintf("Searching for a viable config file in %v", objPaths.ExeDir))
			lstFiles := FindFilesExt(objPaths.ExeDir, ".ini")
			if len(lstFiles) == 0 {
				objLogger.Log("Failed to find any configuration files in the execution directory")
				*strConfFile = GetInput("Please provide a full path to the desired configuration file, or specify env to use environment variables instead: ")
				if *strConfFile != "env" && (*strConfFile == "" || !FileExists(*strConfFile)) {
					objLogger.LogEntry("Can't go on without a valid configuration file", 0, true)
				}
			} else if len(lstFiles) == 1 {
				objLogger.Log(fmt.Sprintf("Found a possible configuration files, do you want %v ?", lstFiles[0]))
				strResponse := GetInput("Type yes to accept, or provide a full path to the desired configuration file, or specify env to use environment variables instead: ")
				if strResponse == "yes" {
					*strConfFile = filepath.Join(objPaths.ExeDir, lstFiles[0])
				} else {
					*strConfFile = strResponse
				}
				if *strConfFile != "env" && (*strConfFile == "" || !FileExists(*strConfFile)) {
					objLogger.LogEntry("Can't go on without a valid configuration file", 0, true)
				}
			} else {
				objLogger.Log("Found few possible configuration files, would any of these work?")
				for i, strEntry := range lstFiles {
					objLogger.Log(fmt.Sprintf("   %d: %s", i+1, strEntry))
				}
				objLogger.Log(fmt.Sprintf("   %d: Provide manually", len(lstFiles)+1))
				objLogger.Log(fmt.Sprintf("   %d: Use environment variables", len(lstFiles)+2))
				objLogger.Log(fmt.Sprintf("   %d: Abort", len(lstFiles)+3))
				strResponse := GetInput("Type the number of your choice: ")
				strInput := strings.TrimSpace(strResponse)
				iChoice, err := strconv.Atoi(strInput)
				if err != nil {
					objLogger.LogEntry(fmt.Sprintf("Invalid selection %v!! Aborting.", strResponse), 0, true)
				}
				objLogger.Log(fmt.Sprintf("You selected %v", iChoice))
				objLogger.LogEntry(fmt.Sprintf("List len: %v", len(lstFiles)), 3, false)

				if iChoice < 1 || iChoice > len(lstFiles)+3 {
					objLogger.LogEntry(fmt.Sprintf("selection %v out of range!! Aborting.", strResponse), 0, true)
				}
				if iChoice == len(lstFiles)+3 {
					objLogger.LogEntry("OK Got it, bailing", 0, true)
				}
				if iChoice == len(lstFiles)+2 {
					*strConfFile = "env"
				}
				if iChoice == len(lstFiles)+1 {
					*strConfFile = GetInput("Please specify full path for your desired config file: ")
					if *strConfFile != "env" && (*strConfFile == "" || !FileExists(*strConfFile)) {
						objLogger.LogEntry("Can't go on without a valid configuration file", 0, true)
					}
				}
				if iChoice < len(lstFiles)+1 {
					*strConfFile = filepath.Join(objPaths.ExeDir, lstFiles[iChoice-1])
					objLogger.Log(fmt.Sprintf("Conf file is now %v", *strConfFile))
				}
				if *strConfFile != "env" && (*strConfFile == "" || !FileExists(*strConfFile)) {
					objLogger.LogEntry("Can't go on without a valid configuration file", 0, true)
				}
			}
		}
	} else {
		*strConfFile = "env"
	}
}
