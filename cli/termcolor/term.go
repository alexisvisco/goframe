package termcolor

import "fmt"

// ANSI color codes
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	White  = "\033[37m"
	Black  = "\033[30m"

	BrightRed    = "\033[91m"
	BrightGreen  = "\033[92m"
	BrightYellow = "\033[93m"
	BrightBlue   = "\033[94m"
	BrightPurple = "\033[95m"
	BrightCyan   = "\033[96m"
	BrightWhite  = "\033[97m"

	BgRed    = "\033[41m"
	BgGreen  = "\033[42m"
	BgYellow = "\033[43m"
	BgBlue   = "\033[44m"
	BgPurple = "\033[45m"
	BgCyan   = "\033[46m"
	BgWhite  = "\033[47m"
	BgBlack  = "\033[40m"
)

// Colorize wraps text with the specified color and automatically resets
func Colorize(color, text string) string {
	return color + text + Reset
}

// Print functions for each color
func WrapRed(text string) string {
	return Colorize(Red, text)
}

func WrapGreen(text string) string {
	return Colorize(Green, text)
}

func WrapYellow(text string) string {
	return Colorize(Yellow, text)
}

func WrapBlue(text string) string {
	return Colorize(Blue, text)
}

func WrapPurple(text string) string {
	return Colorize(Purple, text)
}

func WrapCyan(text string) string {
	return Colorize(Cyan, text)
}

func WrapWhite(text string) string {
	return Colorize(White, text)
}

func WrapBlack(text string) string {
	return Colorize(Black, text)
}

// Printf functions that print directly to stdout
func RedPrintf(format string, args ...interface{}) {
	fmt.Printf(Red+format+Reset, args...)
}

func GreenPrintf(format string, args ...interface{}) {
	fmt.Printf(Green+format+Reset, args...)
}

func YellowPrintf(format string, args ...interface{}) {
	fmt.Printf(Yellow+format+Reset, args...)
}

func BluePrintf(format string, args ...interface{}) {
	fmt.Printf(Blue+format+Reset, args...)
}

func PurplePrintf(format string, args ...interface{}) {
	fmt.Printf(Purple+format+Reset, args...)
}

func CyanPrintf(format string, args ...interface{}) {
	fmt.Printf(Cyan+format+Reset, args...)
}

// Println functions that print with newline
func RedPrintln(text string) {
	fmt.Println(Red + text + Reset)
}

func GreenPrintln(text string) {
	fmt.Println(Green + text + Reset)
}

func YellowPrintln(text string) {
	fmt.Println(Yellow + text + Reset)
}

func BluePrintln(text string) {
	fmt.Println(Blue + text + Reset)
}

func PurplePrintln(text string) {
	fmt.Println(Purple + text + Reset)
}

func CyanPrintln(text string) {
	fmt.Println(Cyan + text + Reset)
}
