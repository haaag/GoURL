package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/atotto/clipboard"
)

const AppName = "go-extract-url"

var (
	browser     string
	copyFlag    bool
	dmenuArgs   string
	openFlag    bool
	verboseFlag bool
)

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

func findURLs(line string) []string {
	urlRegex := `(((http|https|gopher|gemini|ftp|ftps|git)://|www\.)[a-zA-Z0-9.]*[:;a-zA-Z0-9./+@<span class="math-inline">&%?</span>\#=_~-]*)|((magnet:\\?xt=urn:btih:)[a-zA-Z0-9]*)`
	re := regexp.MustCompile(urlRegex)
	return re.FindAllString(line, -1)
}

func copyURL(url string) error {
	err := clipboard.WriteAll(url)
	if err != nil {
		return err
	}

	log.Print("text copied to clipboard: ", url)
	return nil
}

func openURL(url string) error {
	log.Printf("opening URL %s with '%s'\n", url, browser)

	var cmd = exec.Command(browser, url)

	err := cmd.Start()
	if err != nil {
		return err
	}

	return nil
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}

	return def
}

type Menu struct {
	Command   string
	Arguments []string
}

func (m *Menu) Run(s string) (string, error) {
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
		return "", fmt.Errorf("error starting dmenu: %w", err)
	}

	output, err := io.ReadAll(stdoutPipe)
	if err != nil {
		return "", fmt.Errorf("error reading output: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		return "", fmt.Errorf("user hit scape: %w", err)
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
		"-p", "Select URL>",
		"-l", "10",
	},
}

func getURLs() []string {
	var urls []string
	scanner := bufio.NewScanner(os.Stdin)
	index := 1
	for scanner.Scan() {
		line := scanner.Text()
		found := findURLs(line)
		if len(found) > 0 {
			for _, url := range found {
				urls = append(urls, fmt.Sprintf("[%d] %s", index, url))
				index++
			}
		}
	}

	log.Println("found", len(urls), "URLs")
	return urls
}

func selectURL(urls []string) string {
	itemsString := strings.Join(urls, "\n")
	output, err := menu.Run(itemsString)
	if err != nil {
		log.Printf("Error running dmenu: %v\n", err)
		return ""
	}

	selectedStr := strings.Trim(output, "\n")
	if selectedStr == "" {
		log.Println("No URL selected")
		return ""
	}

	return selectedStr
}

func init() {
	browser = getEnv("BROWSER", "xdg-open")
}

func printUsage() {
	fmt.Printf(`Usage: %s [flags]

  Extract URLs from STDIN and show them in dmenu

Optional arguments:
  -c, --copy         copy to clipboard
  -o, --open         open in default browser
  -a, --args         additional args for dmenu
  -v, --verbose      verbose mode
  -h, --help         show this message
`, AppName)
}

func main() {
	flag.BoolVar(&copyFlag, "copy", false, "copy to clipboard")
	flag.BoolVar(&copyFlag, "c", false, "copy to clipboard")

	flag.BoolVar(&openFlag, "open", false, "open in browser")
	flag.BoolVar(&openFlag, "o", false, "open in browser")

	flag.BoolVar(&verboseFlag, "verbose", false, "verbose mode")
	flag.BoolVar(&verboseFlag, "v", false, "verbose mode")

	flag.StringVar(&dmenuArgs, "args", "", "additional args for dmenu")
	flag.Usage = printUsage
	flag.Parse()

	setLoggingLevel(&verboseFlag)

	if !copyFlag && !openFlag {
		flag.Usage()
		return
	}

}
