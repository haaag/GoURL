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
	urlRegex = regexp.MustCompile(
		`(((http|https|gopher|gemini|ftp|ftps|git)://|www\.)[a-zA-Z0-9.]*[:;a-zA-Z0-9./+@&%?$\#=_~-]*)`,
	)
	emailRegex = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)
)

var (
	appName         = "gourl"
	appVersion      = "0.1.5"
	errNoItemsFound = errors.New("no items found")
)

var (
	copyFlag        bool
	customRegexFlag string
	emailFlag       bool
	indexFlag       bool
	limitFlag       int
	menuArgsFlag    string
	openFlag        bool
	promptFlag      string
	urlFlag         bool
	verboseFlag     bool
	versionFlag     bool
)

func usage() {
	fmt.Printf(`%s
Extract URLs from STDIN

Usage: 
  %s [options]

Options:
  -c, --copy        Copy to clipboard
  -o, --open        Open with xdg-open
  -u, --urls        Extract URLs (default: true)
  -e, --email       Extract emails (prefix: "mailto:")
  -E, --regex       Custom regex search
  -l, --limit       Limit number of items
  -i, --index       Add index to URLs found
  -p, --prompt      Prompt for dmenu
  -a, --args        Args for dmenu
  -V, --version     Output version information
  -v, --verbose     Verbose mode
  -h, --help        Show this message
`, version(), appName)
}

// logErrAndExit logs the error and exits the program.
func logErrAndExit(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", appName, err)
		os.Exit(1)
	}
}

// info prints the info to STDOUT.
func info(s string) {
	if !verboseFlag {
		if _, err := fmt.Fprintf(os.Stdout, "%s: %s\n", appName, s); err != nil {
			logErrAndExit(err)
		}

		return
	}

	log.Print(s)
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

// newRegexMatcherWithPrefix returns a function that prepends a prefix to URLs
// matched by the provided regex.
func newRegexMatcherWithPrefix(re *regexp.Regexp, prefix string) func(string) []string {
	return func(line string) []string {
		var (
			matches = re.FindAllString(line, -1)
			urls    = make([]string, 0)
		)

		for _, match := range matches {
			url := strings.Split(match, " ")[0]
			if prefix != "" {
				url = fmt.Sprintf("%s%s", prefix, url)
			}
			urls = append(urls, url)
		}

		return urls
	}
}

// removeIdx returns the input string with the first index identifier removed.
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

// outputData outputs the URLs to STDOUT.
func outputData(d *[]string) {
	for _, url := range *d {
		if _, err := fmt.Fprintln(os.Stdout, url); err != nil {
			logErrAndExit(err)
		}
	}
}

// copyURL copies the selected URL to the clipboard.
func copyURL(url string) error {
	err := clipboard.WriteAll(url)
	if err != nil {
		return fmt.Errorf("error copying to clipboard: %w", err)
	}
	log.Print("text copied to clipboard: ", url)

	return nil
}

// getOSArgs returns the correct arguments for the OS to open with the default
// browser.
func getOSArgs() []string {
	var args []string
	switch runtime.GOOS {
	case "darwin":
		args = []string{"open"}
	case "windows":
		args = []string{"cmd", "/c", "start"}
	default:
		args = []string{"xdg-open"}
	}

	return args
}

// openURL opens the selected URL on the system default browser.
func openURL(url string) error {
	args := getOSArgs()
	args = append(args, url)
	log.Printf("opening URL %s with '%s'\n", url, args)
	cmd := exec.Command(args[0], args[1:]...)
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("error opening URL: %w", err)
	}

	return nil
}

// Menu is a struct that holds the command and arguments for menu.
type Menu struct {
	Command   string
	Arguments []string
}

// prompt sets the prompt for the menu.
func (m *Menu) prompt(s string) {
	m.Arguments = append(m.Arguments, "-p", s)
}

// addArgs adds additional arguments to the menu.
func (m *Menu) addArgs() {
	if menuArgsFlag == "" {
		return
	}

	log.Println("addArgs: ", menuArgsFlag)

	args := strings.Split(menuArgsFlag, " ")
	m.Arguments = append(m.Arguments, args...)
}

// handlePrompt handles the menu prompt.
func (m *Menu) handlePrompt(n int) {
	if promptFlag != "" {
		log.Printf("handlePrompt: promptFlag: '%s'", promptFlag)
		m.prompt(promptFlag)
		return
	}

	s := fmt.Sprintf("%d ", n)
	switch {
	case openFlag:
		s += "Open>"
	case copyFlag:
		s += "Copy>"
	default:
		s += "GoURLs>"
	}

	log.Printf("handlePrompt: string: '%s'", s)

	m.prompt(s)
}

// selection runs the menu command and returns the selected item.
func (m *Menu) selection(s string) (string, error) {
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

// processInputData processes the input from stdin.
func processInputData(s *bufio.Scanner) *[]string {
	var data []string
	for s.Scan() {
		line := s.Text()
		data = append(data, line)
	}
	if err := s.Err(); err != nil {
		logErrAndExit(err)
	}

	log.Printf("processInputData: input data: %d\n", len(data))

	return &data
}

// uniqueItems removes duplicates from a slice.
func uniqueItems(d *[]string) {
	seen := make(map[string]bool)
	var result []string
	for _, ok := range *d {
		if !seen[ok] {
			seen[ok] = true
			result = append(result, ok)
		}
	}

	log.Printf("uniqueItems: result: %d\n", len(result))

	*d = result
}

// addIndex adds an index to the items.
func addIndex(d *[]string) {
	if !indexFlag {
		return
	}

	r := make([]string, len(*d))
	for i, url := range *d {
		r[i] = fmt.Sprintf("[%d] %s", i+1, url)
	}

	log.Printf("addIndex: result: %d\n", len(r))

	*d = r
}

// scanItems scans the input data and returns the found match.
func scanItems(d *[]string, find func(string) []string) []string {
	var items []string
	for _, line := range *d {
		found := find(line)
		if len(found) == 0 {
			continue
		}

		items = append(items, found...)

		// limit the number of items.
		if len(items) >= limitFlag {
			log.Printf("scanItems: breaking after '%d' items\n", limitFlag)
			break
		}
	}

	log.Printf("scanItems: result: %d\n", len(items))

	return items
}

// scanURLs scans the input data and returns the found URLs.
func scanURLs(d *[]string, find func(string) []string, resultsCh chan []string) {
	log.Println("scanURLs: scanning...")
	items := scanItems(d, find)
	resultsCh <- items
}

// getURLsFrom concurrently searches for URLs using multiple finders and stores
// the results in the provided slice.
func getURLsFrom(d *[]string, finders ...func(string) []string) error {
	resultsCh := make(chan []string)

	if limitFlag == 0 {
		limitFlag = len(*d)
	}

	// Start finders.
	for _, f := range finders {
		go scanURLs(d, f, resultsCh)
	}

	results := make([]string, 0)
	// Wait for all finders to finish.
	for range finders {
		results = append(results, <-resultsCh...)
	}

	if len(results) == 0 {
		log.Println("getURLsFrom: no items found")
		return errNoItemsFound
	}

	log.Printf("getURLsFrom: result: %d\n", len(results))

	*d = results

	return nil
}

// selectURL runs menu and returns the selected URL.
func selectURL(d *[]string) string {
	itemsString := strings.Join(*d, "\n")
	output, err := menu.selection(itemsString)
	if err != nil {
		return ""
	}

	selectedStr := strings.Trim(output, "\n")
	if selectedStr == "" {
		info("no <item> selected")
		return ""
	}

	return selectedStr
}

// handleURLAction executes an action on a URL based on enabled flags.
func handleURLAction(url string) {
	actions := map[bool]func(url string) error{
		copyFlag: copyURL,
		openFlag: openURL,
	}

	if action, ok := actions[true]; ok {
		logErrAndExit(action(removeIdx(url)))
		os.Exit(0)
	}
}

// findWithCustomRegex finds URLs based on the custom regex.
func findWithCustomRegex(d *[]string) error {
	if customRegexFlag == "" {
		return nil
	}

	log.Printf("findWithCustomRegex: regex: '%s'\n", customRegexFlag)

	r := regexp.MustCompile(customRegexFlag)
	matcher := newRegexMatcherWithPrefix(r, "")

	return getURLsFrom(d, matcher)
}

// findItems finds items (urls or emails) in the provided slice based on
// enabled flags and custom regex.
func findItems(d *[]string) error {
	if customRegexFlag != "" {
		return nil
	}

	var finders []func(string) []string

	log.Printf("findItems: urlFlag: %t\n", urlFlag)
	log.Printf("findItems: emailFlag: %t\n", emailFlag)

	if urlFlag {
		// append <find URLs> function.
		finders = append(finders, newRegexMatcherWithPrefix(urlRegex, ""))
	}

	if emailFlag {
		// append <find emails> function.
		finders = append(finders, newRegexMatcherWithPrefix(emailRegex, "mailto:"))
	}

	log.Printf("findItems: finders functions: %d\n", len(finders))

	return getURLsFrom(d, finders...)
}

// handleItems processes items (urls) based on enabled flags and user input.
func handleItems(d *[]string) {
	// If no action flags are passed, just print the items found.
	if !copyFlag && !openFlag && menuArgsFlag == "" {
		log.Println("no action flags passed, printing items:")
		outputData(d)

		return
	}

	n := len(*d)

	menu.addArgs()
	menu.handlePrompt(n)

	url := selectURL(d)
	if url == "" {
		return
	}

	handleURLAction(url)
}

func version() string {
	return fmt.Sprintf("%s v%s %s/%s\n", appName, appVersion, runtime.GOOS, runtime.GOARCH)
}

func init() {
	flag.BoolVar(&copyFlag, "c", false, "copy to clipboard")
	flag.BoolVar(&copyFlag, "copy", false, "copy to clipboard")

	flag.BoolVar(&openFlag, "o", false, "open in browser")
	flag.BoolVar(&openFlag, "open", false, "open in browser")

	flag.BoolVar(&emailFlag, "e", false, "extract emails")
	flag.BoolVar(&emailFlag, "email", false, "extract emails")

	flag.BoolVar(&urlFlag, "u", true, "extract URLs")
	flag.BoolVar(&urlFlag, "url", true, "extract URLs")

	flag.IntVar(&limitFlag, "l", 0, "limit number of URLs")
	flag.IntVar(&limitFlag, "limit", 0, "limit number of URLs")

	flag.BoolVar(&verboseFlag, "v", false, "verbose mode")
	flag.BoolVar(&verboseFlag, "verbose", false, "verbose mode")

	flag.BoolVar(&indexFlag, "i", false, "indexed menu")
	flag.BoolVar(&indexFlag, "index", false, "indexed menu")

	flag.StringVar(&customRegexFlag, "E", "", "custom regex")
	flag.StringVar(&customRegexFlag, "regex", "", "custom regex")

	flag.StringVar(&menuArgsFlag, "a", "", "additional args for dmenu")
	flag.StringVar(&menuArgsFlag, "args", "", "additional args for dmenu")

	flag.StringVar(&promptFlag, "p", "", "prompt for dmenu")
	flag.StringVar(&promptFlag, "prompt", "", "prompt for dmenu")

	flag.BoolVar(&versionFlag, "V", false, "output version information")
	flag.BoolVar(&versionFlag, "version", false, "output version information")

	flag.Usage = usage
	flag.Parse()

	if versionFlag {
		fmt.Print(version())
		os.Exit(0)
	}

	setVerboseLevel()
}

func main() {
	s := bufio.NewScanner(os.Stdin)
	d := processInputData(s)

	if err := findWithCustomRegex(d); err != nil {
		logErrAndExit(err)
	}

	if err := findItems(d); err != nil {
		logErrAndExit(err)
	}

	uniqueItems(d)
	addIndex(d)

	handleItems(d)
}
