# Whodunnit

Count and blame your repo.

## About

Whodunnit is a command-line TUI that counts and categorizes the number of lines in a repo by filetype and contributor. It was originally created to gather data about a class project, and expanded on to be more useable and useful.

## Installation

_Whodunnit is currently in prerelease._

You can download the latest release of Whodunnit from the [releases](https://github.com/connorgannaway/whodunnit/releases) page, or through:

```bash
# go
go install github.com/connorgannaway/whodunnit@v0

# homebrew
brew tap connorgannaway/whodunnit
brew install whodunnit

# ubuntu
echo "deb [trusted=yes] https://apt.fury.io/connorgannaway/ /" | sudo tee /etc/apt/sources.list.d/fury-connorgannaway.list > /dev/null
sudo apt update
sudo apt install whodunnit
```

## Usage

Run `whodunnit` without arguments in a directory you wish to count or (optionally) pass a directory to target as the first argument.

## Roadmap

|  #  | Feature                            | Status |
| :-: | ---------------------------------- | :----: |
|  1  | Sorting by count, filetype, author |   ❌   |
|  2  | JSON Export                        |   ❌   |
|  3  | Filtering included file types      |   ❌   |
|  4  | Filtering by date range            |   ❌   |
