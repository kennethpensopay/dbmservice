package godebug

import (
	"fmt"
	"log"
)

var enabled = false

const (
	// BoldFont Set font weight to Bold
	BoldFont = "\033[1m"
	// GreenFont Set font color to Green
	GreenFont = "\033[92m"
	// YellowFont Set font color to Yellow
	YellowFont = "\033[93m"
	// RedFont Set font color to Red
	RedFont = "\033[91m"
	// ResetFont Remove font formatting
	ResetFont = "\033[0m"
)

func EnableDebug() {
	enabled = true
}

func DisableDebug() {
	enabled = false
}

func Debugf(format string, a ...any) {
	Debug(fmt.Sprintf(format, a...))
}

func Debug(message string) {
	if enabled {
		log.Printf("%s%s[DEBUG]:%s %s%s\n", BoldFont, YellowFont, RedFont, message, ResetFont)
	}
}
