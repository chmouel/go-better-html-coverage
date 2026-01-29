# go-better-html-coverage

Generates a single HTML file from Go coverage profiles. No external dependencies, works offline.

## Install

```bash
go install github.com/chmouel/go-better-html-coverage@latest
```

## Usage

```bash
go test -coverprofile=coverage.out ./...
go-better-html-coverage -profile coverage.out -o coverage.html
```

Open `coverage.html` in a browser.

## Flags

- `-profile` - path to coverage profile (default: `coverage.out`)
- `-o` - output HTML file (default: `coverage.html`)
- `-src` - source root directory (default: `.`)

## Shortcuts

- `Ctrl+P` - focus file search
- `Ctrl+F` - search in current file
- `Enter` / `Shift+Enter` - next/previous match

## Licence

MIT
