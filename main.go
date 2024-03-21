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
	testFlag     bool
	verboseFlag  bool
	xdgOpen      string
	versionFlag  bool
)

type finderFunc func(string) []string

func printUsage() {
	fmt.Printf(`Usage: %s [flags]

  Extract URLs from STDIN

Optional arguments:
  -c, --copy          Copy to clipboard
  -o, --open          Open in default browser (xdg-open)
  -l, --limit         Limit number of URLs
  -i, --index         Add index to URLs found
  -a, --menu-args     Additional args for dmenu
  -v, --verbose       Verbose mode
  -h, --help          Show this message
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

// setLoggingLevel sets the logging level based on the verbose flag.
func setLoggingLevel(verboseFlag *bool) {
	if *verboseFlag {
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

func outputData(urls []string) {
	for _, url := range urls {
		fmt.Fprintln(os.Stdout, url)
	}
}

// copyURL copies the selected URL to the clipboard
func copyURL(url string) error {
	err := clipboard.WriteAll(url)
	if err != nil {
		return err
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
		return err
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
func (m *Menu) addArgs(menuArgs string) {
	if menuArgs == "" {
		return
	}

	args := strings.Split(menuArgs, " ")
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
		return "", err
	}

	outputStr := string(output)
	outputStr = strings.TrimRight(outputStr, "\n")
	log.Println("selected:", outputStr)

	return outputStr, nil
}

var dmenu = Menu{
	Command: "dmenu",
	Arguments: []string{
		"-i",
		"-l", "10",
	},
}

func processInputData(r io.Reader, limit int) []string {
	var data []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if limit > 0 && len(data) >= limit {
			break
		}
		line := scanner.Text()
		data = append(data, line)
	}
	return data
}

func scanURLs(data []string, indexed bool, find finderFunc, results chan []string) {
	var items []string
	index := 1
	for _, line := range data {
		found := find(line)
		if len(found) > 0 {
			for _, item := range found {
				if indexed {
					items = append(items, fmt.Sprintf("[%d] %s", index, item))
				} else {
					items = append(items, item)
				}
				index++
			}
		}
	}
	results <- items
}

func getURLsFrom(r io.Reader) ([]string, error) {
	resultsCh := make(chan []string)

	data := processInputData(r, limitFlag)
	go scanURLs(data, indexFlag, findURLs, resultsCh)
	go scanURLs(data, indexFlag, findEmails, resultsCh)
	emails := <-resultsCh
	urls := <-resultsCh

	allURLs := mergeSlices(emails, urls)
	if len(allURLs) == 0 {
		return nil, errNoURLFound
	}

	return allURLs, nil
}

// selectURL runs menu and returns the selected URL
func selectURL(urls []string, menu *Menu) string {
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
		logErrAndExit(action(url))
		os.Exit(0)
	} else {
		// No action, just print
		fmt.Println(url)
	}
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

	flag.BoolVar(&versionFlag, "version", false, "version info")

	flag.BoolVar(&testFlag, "test", false, "find emails")
	flag.Usage = printUsage
	flag.Parse()

	if versionFlag {
		fmt.Print(version())
		os.Exit(0)
	}

	setLoggingLevel(&verboseFlag)
}

func main() {
	if flag.NFlag() == 0 && flag.NArg() == 0 {
		flag.Usage()
		return
	}
	setLoggingLevel(&verboseFlag)

	dmenu.addArgs(menuArgsFlag)
	dmenu.handlePrompt()

	urls := getURLs(limitFlag, indexFlag)
	if len(urls) == 0 {
		logErrAndExit(fmt.Errorf("no <URLs> found"))
	}

	if dumpFileFlag != "" {
		if err := dumpURLs(urls, dumpFileFlag); err != nil {
			logErrAndExit(err)
		}
		os.Exit(0)
	}

	url := selectItem(urls)
	if url == "" {
		return
	}

	handleURLAction(url)
}
