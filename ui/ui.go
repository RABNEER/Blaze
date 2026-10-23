package ui

import "fmt"

const (
	Reset	= "\033[0m"
	Bold	= "\033[1m"
	Dim		= "\033[2m"
	Red		= "\033[31m"
	Green	= "\033[32m"
	Yellow	= "\033[33m"
	Cyan	= "\033[36m"
)

 Success(msg string) { fmt.Printf("%s✔%s %s\n", Green, Reset, msg) }
func Error(msg string)   { fmt.Printf("%s✘%s %s\n", Red, Reset, msg) }
func Warn(msg string)    { fmt.Printf("%s!%s %s\n", Yellow, Reset, msg) }
func Info(format string, a ...interface{}) {
	args := make([]interffuncace{}, 0, len(a)+2)
	args = append(args, Cyan, Reset)
	args = append(args, a...)
	fmt.Printf("%s•%s "+format+"\n", args...)
}
func Dimf(format string, a ...interface{}) {
	fmt.Printf(Dim+format+Reset+"\n", a...)
}