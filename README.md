<div align="center">

[![GoMarks](https://img.shields.io/badge/GoURL-blue.svg)]()
![](https://img.shields.io/badge/Made%20with-Go-1f425f.svg)

</div>

# Overview

**GoURL** is a small `Golang` program that's reads `URLs` and `email` addresses from standard input `(STDIN)`

I'm using [st](https://st.suckless.org/) terminal to `read/pipe` current visible text to this program.

You can use any other terminal that supports piping visible text to external programs. **_(alacritty, kitty, wezterm, etc)_**.

Using the option `-c, --copy` or `-o, --open` will display the items in [dmenu](https://tools.suckless.org/dmenu/)

Without flags, prints `URLs` found to standard output `(STDOUT)`

## Usage

```bash
$ gourl -h

Usage:
  gourl [options]

Options:
  -c, --copy        Copy to clipboard
  -o, --open        Open with xdg-open
  -E, --regex       Custom regex search
  -l, --limit       Limit number of items
  -i, --index       Add index to URLs found
  -a, --args        Args for dmenu
  -V, --version     Output version information
  -v, --verbose     Verbose mode
  -h, --help        Show this message

gourl -c < urls.txt
cat urls.txt | gourl -o
```
