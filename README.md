
<div align="center">
    <h1><b>🔗 GoURL</b></h1>
    <sub>✨ do <b>not</b> use <b>regex</b> 🤡</sub>
<br>
<br>

[![MadeWithGo](https://img.shields.io/badge/Made%20with-Go-1f425f.svg)](https://go.dev/)
</div>


**GoURL** is a small `Golang` program that reads `URLs` and `email` addresses from standard input `(STDIN)`

You can use any **terminal** that supports piping visible text to external programs **_(alacritty, kitty, wezterm, etc)_**.

I'm using [st](https://st.suckless.org/) terminal with [externalpipe](https://st.suckless.org/patches/externalpipe/) patch to `read/pipe` current visible text to this program.

Using the option `-c, --copy` or `-o, --open` will display the items in [dmenu](https://tools.suckless.org/dmenu/)

Without flags, prints `URLs` found to standard output `(STDOUT)`, **_you can pipe it to your preferred menu or launcher_**.

### ✨ Features

- Extract URLs from `STDIN`
- Choose items with `dmenu`
- Ignore `duplicates`
- Copy to clipboard
- Open with `xdg-open`
- Custom regex search
- Add `index` to URLs found
- Limit number of items

### ⚡️Requirements

- [Go](https://golang.org/) `v1.21.3` or higher

### 📦 Build

```bash
# clone the repo
$ git clone 'https://github.com/haaag/GoURL' && cd GoURL

# use make to build
$ make
```

<sub>Binary can be found in `GoURL/bin`.</sub>

### 📦 Installation

```bash
# use make to build
$ make

# install on system wide
$ sudo make install

# or use a symlink
$ ln -sf $PWD/bin/gourl ~/.local/bin/
```

<sub>Uninstall from system, use <b>sudo make uninstall</b> or remove symlink with `rm ~/.local/bin/gourl`</sub>

### 🚀 Usage

```bash
$ gourl -h
Extract URLs from STDIN

Usage:
  gourl [options]

Options:
  -c, --copy        Copy to clipboard
  -o, --open        Open with xdg-open
  -e, --email       Extract emails
  -E, --regex       Custom regex search
  -l, --limit       Limit number of items
  -i, --index       Add index to URLs found
  -a, --args        Args for dmenu
  -V, --version     Output version information
  -v, --verbose     Verbose mode
  -h, --help        Show this message

$ gourl -c < urls.txt
$ cat urls.txt | gourl -c
```

### 🚩 Using `-E` flag

The flag `-E` can be use for `custom regex`, like in `grep`.

```bash
# list existing remotes
git remote -v | gourl -E '((git|ssh|http(s)?)|(git@[\w\.]+))(:(//)?)([\w\.@\:/\-~]+)(\.git)(/)?'
```

### ⭐ Related projects

- [urlscan](https://github.com/firecat53/urlscan) - Designed to integrate with the "mutt" mailreader
- [urlview](https://github.com/sigpipe/urlview) - Extract URLs from a text file and allow the user to select via a menu
