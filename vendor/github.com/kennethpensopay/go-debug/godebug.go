package godebug

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

var debugEnabled = false

const (
	// Bold Set font weight to Bold
	Bold = "\033[1m"
	// foregroundGreen Set font color to Green
	foregroundGreen = "\033[92m"
	// foregroundYellow Set font color to Yellow
	foregroundYellow = "\033[93m"
	// foregroundRed Set font color to Red
	foregroundRed = "\033[91m"
	// ClearFont Remove font formatting
	ClearFont = "\033[0m"
)

func init() {
	log.Println("Initializing dbmdebug ...")
	if d, err := strconv.ParseBool(strings.TrimSpace(os.Getenv("DEBUG"))); err == nil {
		debugEnabled = d
	}
	Debug("Debug is enabled.")
}

func Debugf(format string, a ...any) {
	Debug(fmt.Sprintf(format, a...))
}

func Debug(message string) {
	if debugEnabled {
		log.Printf("%s%s[DEBUG]:%s %s%s\n", Bold, foregroundYellow, foregroundRed, message, ClearFont)
	}
}
