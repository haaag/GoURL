// GoURL
//
// Extract URLs from STDIN
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/atotto/clipboard"
)

var (
	appName       = "gourl"
	appVersion    = "0.0.1"
	errNoURLFound = errors.New("no urls found")
)

var (
	copyFlag     bool
	openFlag     bool
	limitFlag    int
	indexFlag    bool
	menuArgsFlag string
	verboseFlag  bool
	xdgOpen      string
	versionFlag  bool
)

func printUsage() {
	fmt.Printf(`Extract URLs from STDIN

Usage: 
  %s [options]

Options:
  -c, --copy       copy to clipboard
  -o, --open       open in xdg-open
  -l, --limit      limit number of URLs
  -i, --index      add index to URLs found
  -a, --menu-args  additional args for dmenu
  -V, --version    output version information
  -v, --verbose    verbose mode
  -h, --help       show this message
`, appName)
}

// logErrAndExit logs the error and exits the program
func logErrAndExit(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", appName, err)
		os.Exit(1)
	}
}

func printInfo(s string) {
	if !verboseFlag {
		fmt.Fprintf(os.Stdout, "%s: %s\n", appName, s)
	} else {
		log.Print(s)
	}
}

// setVerboseLevel sets the logging level based on the verbose flag.
func setVerboseLevel() {
	if verboseFlag {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Println("verbose mode: on")

		return
	}

	silentLogger := log.New(io.Discard, "", 0)
	log.SetOutput(silentLogger.Writer())
}

// findEmails finds all emails in a string
func findEmails(line string) []string {
	emailRegex := `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`
	re := regexp.MustCompile(emailRegex)
	matches := re.FindAllString(line, -1)
	emails := make([]string, 0)
	for _, match := range matches {
		email := strings.Split(match, " ")[0]
		emails = append(emails, fmt.Sprintf("mailto:%s", email))
	}
	return emails
}

// findURLs finds all URLs in a string
func findURLs(line string) []string {
	urlRegex := `(((http|https|gopher|gemini|ftp|ftps|git)://|www\\.)[a-zA-Z0-9.]*[:;a-zA-Z0-9./+@$&%?$\#=_~-]*)|((magnet:\\?xt=urn:btih:)[a-zA-Z0-9]*)`
	re := regexp.MustCompile(urlRegex)
	matches := re.FindAllString(line, -1)
	urls := make([]string, 0)
	for _, match := range matches {
		url := strings.Split(match, " ")[0]
		urls = append(urls, url)
	}
	return urls
}

func removeIdx(s string) string {
	if !indexFlag {
		return s
	}

	split := strings.Split(s, " ")
	if len(split) == 1 {
		return s
	}

	return split[1]
}

// outputData outputs the URLs to STDOUT
func outputData(urls []string) {
	for _, url := range urls {
		fmt.Fprintln(os.Stdout, url)
	}
}

// copyURL copies the selected URL to the clipboard
func copyURL(url string) error {
	err := clipboard.WriteAll(url)
	if err != nil {
		return fmt.Errorf("error copying to clipboard: %w", err)
	}

	log.Print("text copied to clipboard: ", url)
	return nil
}

// openURL opens the selected URL in the
func openURL(url string) error {
	log.Printf("opening URL %s with '%s'\n", url, xdgOpen)

	cmd := exec.Command(xdgOpen, url)

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("error opening URL: %w", err)
	}

	return nil
}

// Menu is a struct that holds the command and arguments for menu
type Menu struct {
	Command   string
	Arguments []string
}

// prompt sets the prompt for the menu
func (m *Menu) prompt(s string) {
	m.Arguments = append(m.Arguments, "-p", s)
}

// addArgs adds additional arguments to the menu
func (m *Menu) addArgs() {
	if menuArgsFlag == "" {
		return
	}

	args := strings.Split(menuArgsFlag, " ")
	m.Arguments = append(m.Arguments, args...)
}

// handlePrompt handles the menu prompt
func (m *Menu) handlePrompt() {
	switch {
	case openFlag:
		m.prompt("OpenURL>")
	case copyFlag:
		m.prompt("CopyURL>")
	default:
		m.prompt("GoURLs>")
	}
}

// show runs the menu command and returns the selected item
func (m *Menu) show(s string) (string, error) {
	log.Println("running menu:", m.Command, m.Arguments)
	cmd := exec.Command(m.Command, m.Arguments...)

	if s != "" {
		cmd.Stdin = strings.NewReader(s)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("error creating output pipe: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return "", fmt.Errorf("error starting menu: %w", err)
	}

	output, err := io.ReadAll(stdoutPipe)
	if err != nil {
		return "", fmt.Errorf("error reading output: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		return "", fmt.Errorf("error waiting for menu: %w", err)
	}

	outputStr := string(output)
	outputStr = strings.TrimRight(outputStr, "\n")
	log.Println("selected:", outputStr)

	return outputStr, nil
}

var menu = Menu{
	Command: "dmenu",
	Arguments: []string{
		"-i",
		"-l", "10",
	},
}

// processInputData processes the input from stdin
func processInputData(r io.Reader) []string {
	var data []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if limitFlag > 0 && len(data) >= limitFlag {
			break
		}
		line := scanner.Text()
		data = append(data, line)
	}
	return data
}

// scanItems scans the input data and returns the found match
func scanItems(data []string, find func(string) []string) []string {
	var items []string
	index := 1
	for _, line := range data {
		found := find(line)
		for _, item := range found {
			if indexFlag {
				item = fmt.Sprintf("[%d] %s", index, item)
			}
			items = append(items, item)
			index++
		}
	}
	return items
}

// scanURLs scans the input data and returns the found URLs
func scanURLs(data []string, find func(string) []string, resultsCh chan []string) {
	items := scanItems(data, find)
	resultsCh <- items
}

func getURLsFrom(r io.Reader) ([]string, error) {
	resultsCh := make(chan []string)

	data := processInputData(r)
	go scanURLs(data, findURLs, resultsCh)
	go scanURLs(data, findEmails, resultsCh)
	emails := <-resultsCh
	urls := <-resultsCh

	if len(urls) == 0 && len(emails) == 0 {
		return nil, errNoURLFound
	}

	allURLs := mergeSlices(emails, urls)
	return allURLs, nil
}

// selectURL runs menu and returns the selected URL
func selectURL(urls []string) string {
	itemsString := strings.Join(urls, "\n")
	output, err := menu.show(itemsString)
	if err != nil {
		return ""
	}

	selectedStr := strings.Trim(output, "\n")
	if selectedStr == "" {
		printInfo("no <URL> selected")
		return ""
	}

	return selectedStr
}

func handleURLAction(url string) {
	actions := map[bool]func(url string) error{
		copyFlag: copyURL,
		openFlag: openURL,
	}

	if action, ok := actions[true]; ok {
		logErrAndExit(action(removeIdx(url)))
		os.Exit(0)
	}

	// No action, just output
	fmt.Println(url)
}

func mergeSlices(data ...[]string) []string {
	var merged []string
	for _, d := range data {
		merged = append(merged, d...)
	}
	return merged
}

func version() string {
	return fmt.Sprintf("%s v%s %s/%s\n", appName, appVersion, runtime.GOOS, runtime.GOARCH)
}

func init() {
	xdgOpen = "xdg-open"

	flag.BoolVar(&copyFlag, "c", false, "copy to clipboard")
	flag.BoolVar(&copyFlag, "copy", false, "copy to clipboard")

	flag.BoolVar(&openFlag, "o", false, "open in browser")
	flag.BoolVar(&openFlag, "open", false, "open in browser")

	flag.IntVar(&limitFlag, "l", 0, "limit number of URLs")
	flag.IntVar(&limitFlag, "limit", 0, "limit number of URLs")

	flag.BoolVar(&verboseFlag, "v", false, "verbose mode")
	flag.BoolVar(&verboseFlag, "verbose", false, "verbose mode")

	flag.BoolVar(&indexFlag, "i", false, "indexed menu")
	flag.BoolVar(&indexFlag, "index", false, "indexed menu")

	flag.StringVar(&menuArgsFlag, "a", "", "additional args for dmenu")
	flag.StringVar(&menuArgsFlag, "menu-args", "", "additional args for dmenu")

	flag.BoolVar(&versionFlag, "V", false, "output version information")
	flag.BoolVar(&versionFlag, "version", false, "output version information")

	flag.Usage = printUsage
	flag.Parse()

	if versionFlag {
		fmt.Print(version())
		os.Exit(0)
	}

	setVerboseLevel()
}

func main() {
	urls, err := getURLsFrom(os.Stdin)
	if err != nil {
		logErrAndExit(err)
	}

	// If no action flags are passed, just print the URLs
	if !copyFlag && !openFlag && menuArgsFlag == "" {
		outputData(urls)
		return
	}

	menu.addArgs()
	menu.handlePrompt()

	url := selectURL(urls)
	if url == "" {
		return
	}

	handleURLAction(url)
}
