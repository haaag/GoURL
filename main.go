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
	browser      string
	copyFlag     bool
	dumpFileFlag string
	indexFlag    bool
	limitFlag    int
	menuArgsFlag string
	openFlag     bool
	printFlag    bool
	verboseFlag  bool
	versionFlag  bool
)

func printUsage() {
	fmt.Printf(`Usage: %s [flags]

  Extract URLs from STDIN

Optional arguments:
  -c, --copy          Copy to clipboard
  -o, --open          Open in default browser (xdg-open)
  -p, --print         Print selected URL 
  -l, --limit         Limit number of URLs
  -i, --index         Add index to URLs found
  -m, --menu          Show URLs/emails in menu
  -a, --menu-args     Additional args for dmenu
  -s, --save          Save found URLs to <file>
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

// printURL prints the selected URL
func printURL(url string) error {
	fmt.Println(url)
	return nil
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

// dumpURLs dumps the URLs to local file
func dumpURLs(urls []string, filename string) error {
	if filename == "" {
		return nil
	}

	if len(urls) == 0 {
		return nil
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	for _, url := range urls {
		if _, err := f.WriteString(url + "\n"); err != nil {
			return err
		}
	}

	defer f.Close()

	printInfo(fmt.Sprintf("dumping %d URLs to '%s'", len(urls), filename))
	return nil
}

// openURL opens the selected URL in the default browser
func openURL(url string) error {
	log.Printf("opening URL %s with '%s'\n", url, browser)

	var cmd = exec.Command(browser, url)

	err := cmd.Start()
	if err != nil {
		return err
	}

	return nil
}

// getEnv gets an environment variable from the environment,
// falling back to a default value if it is not set
func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}

	return def
}

// Menu is a struct that holds the command and arguments for menu
type Menu struct {
	Command   string
	Arguments []string
}

// prompt sets the prompt for the menu
func (m *Menu) prompt(s string) *Menu {
	m.Arguments = append(m.Arguments, "-p", s)
	return m
}

// addArgs adds additional arguments to the menu
func (m *Menu) addArgs(menuArgs string) *Menu {
	if menuArgs == "" {
		return m
	}

	args := strings.Split(menuArgs, " ")
	m.Arguments = append(m.Arguments, args...)
	return m
}

// handlePrompt handles the menu prompt
func (m *Menu) handlePrompt() {
	switch {
	case copyFlag:
		m.prompt("CopyURL>")
	case openFlag:
		m.prompt("OpenURL>")
	case printFlag:
		m.prompt("PrintURL>")
	default:
		m.prompt("GoURL>")
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

// getURLs gets all URLs from STDIN
func getURLs(limit int, indexed bool) []string {
	var urls []string
	scanner := bufio.NewScanner(os.Stdin)
	index := 1

	for scanner.Scan() {
		if limit > 0 && len(urls) >= limit {
			break
		}
		line := scanner.Text()
		found := findURLs(line)
		if len(found) > 0 {
			for _, url := range found {
				if indexed {
					urls = append(urls, fmt.Sprintf("[%d] %s", index, url))
				} else {
					urls = append(urls, url)
				}
				index++
			}
		}
	}

	log.Println("found", len(urls), "URLs")
	return urls
}

// selectItem runs menu and returns the selected URL
func selectItem(urls []string) string {
	itemsString := strings.Join(urls, "\n")
	output, err := dmenu.show(itemsString)
	if err != nil {
		log.Printf("error running menu: %v\n", err)
		return ""
	}

	selectedStr := strings.Trim(output, "\n")
	if selectedStr == "" {
		printInfo("no <URL> selected")
		return ""
	}

	return selectedStr
}

func printUsage() {
	fmt.Printf(`Usage: %s [flags]

  Extract URLs from STDIN

Optional arguments:
  -c, --copy        copy to clipboard
  -o, --open        open in default browser
  -a, --args        additional args for dmenu
  -p, --print       print selected URL 
  -l, --limit       limit number of URLs
  -d, --dump        dumps URLs to local <file>
  -v, --verbose     verbose mode
  -h, --help        show this message
`, appName)
}

func handleURLAction(url string) {
	if indexFlag {
		split := strings.Split(url, " ")
		if len(split) > 1 {
			url = split[1]
		}
	}

	actions := map[bool]func(url string) error{
		copyFlag:  copyURL,
		openFlag:  openURL,
		printFlag: printURL,
	}

	if action, ok := actions[true]; ok {
		logErrAndExit(action(url))
		os.Exit(0)
	}
	return merged
}

func version() string {
	return fmt.Sprintf("%s v%s %s/%s\n", appName, appVersion, runtime.GOOS, runtime.GOARCH)
}

func init() {
	browser = getEnv("BROWSER", "xdg-open")

	flag.BoolVar(&copyFlag, "copy", false, "copy to clipboard")
	flag.BoolVar(&copyFlag, "c", false, "copy to clipboard")

	flag.BoolVar(&openFlag, "open", false, "open in browser")
	flag.BoolVar(&openFlag, "o", false, "open in browser")

	flag.BoolVar(&printFlag, "print", false, "print selected URL")
	flag.BoolVar(&printFlag, "p", false, "print selected URL")

	flag.IntVar(&limitFlag, "limit", 0, "limit number of URLs")
	flag.IntVar(&limitFlag, "l", 0, "limit number of URLs")

	flag.BoolVar(&indexFlag, "index", false, "indexed menu")

	flag.BoolVar(&verboseFlag, "verbose", false, "verbose mode")
	flag.BoolVar(&verboseFlag, "v", false, "verbose mode")

	flag.BoolVar(&versionFlag, "version", false, "version info")

	flag.StringVar(&menuArgsFlag, "args", "", "additional args for dmenu")
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
