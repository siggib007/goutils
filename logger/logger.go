package logger

import (
	"fmt"
	"os"
	"time"
)

type objAbortSignal struct {
	iExitCode int
}

// Logger handles logging to file and stdout with verbosity levels
type Logger struct {
	IVerbose  int
	ObjLogOut *os.File
	FnUILog   func(string)
}

// NewLogger creates a new Logger writing to strLogFile at the given verbosity level
func NewLogger(strLogFile string, iVerbose int) (*Logger, error) {
	objLogOut, err := os.OpenFile(strLogFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{
		IVerbose:  iVerbose,
		ObjLogOut: objLogOut,
	}, nil
}

// LogEntry logs a message if verbosity level permits
// bAbort causes the program to exit with code 9 after logging
func (l *Logger) LogEntry(strMsg string, iMsgLevel int, bAbort bool) {
	strTimeStamp := time.Now().Format("01-02-2006 15:04:05")
	if l.IVerbose > iMsgLevel {
		fmt.Fprintf(l.ObjLogOut, "%s : %s\n", strTimeStamp, strMsg)
		fmt.Println(strMsg)
		if l.FnUILog != nil {
			l.FnUILog(strMsg)
		}
	} else if bAbort {
		fmt.Fprintf(l.ObjLogOut, "%s : %s\n", strTimeStamp, strMsg)
	}
	if bAbort {
		l.Close()
		panic(&objAbortSignal{iExitCode: 9})
	}
}

// Log logs a message at verbosity level 0
func (l *Logger) Log(strMsg string) {
	l.LogEntry(strMsg, 0, false)
}

// Close closes the log file
func (l *Logger) Close() {
	l.ObjLogOut.Close()
	fmt.Println("objLogOut closed")
}

func (l *Logger) RecoverAbort() {
	objRecovered := recover()
	if objRecovered == nil {
		return
	}

	objAbort, bIsAbortSignal := objRecovered.(*objAbortSignal)
	if !bIsAbortSignal {
		strMsg := fmt.Sprintf("unexpected panic: %v", objRecovered)
		l.LogEntry(strMsg, 0, false)
		l.Close()
		os.Exit(5)
	}

	os.Exit(objAbort.iExitCode)
}
