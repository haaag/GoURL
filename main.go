package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"

	"github.com/atotto/clipboard"
)

const AppName = "go-extract-url"

var (
	browser   string
	copyFlag  bool
	dmenuArgs string
	openFlag  bool
)

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
  -h, --help         show this message
`, AppName)
}

func main() {
	flag.BoolVar(&copyFlag, "copy", false, "copy to clipboard")
	flag.BoolVar(&copyFlag, "c", false, "copy to clipboard")
	flag.BoolVar(&openFlag, "open", false, "open in browser")
	flag.BoolVar(&openFlag, "o", false, "open in browser")
	flag.StringVar(&dmenuArgs, "args", "", "additional args for dmenu")
	flag.Usage = printUsage
	flag.Parse()
}
