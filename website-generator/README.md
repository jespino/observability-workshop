# Website Generator

This Go program generates the workshop website from the markdown module files.

## Features

- Converts markdown modules to HTML pages
- Generates an index page with a module overview
- Builds the English and Spanish sites from one run, with a language switcher
- Includes CSS styling
- Automatic navigation links (previous/next)
- Preserves all markdown formatting and code blocks
- Fixes relative links to work in HTML format
- Optional dev server with live reload

## Usage

### Basic Usage

```bash
# From the website-generator directory
go run . -modules ../modules -output ../website
```

### Installation

```bash
# Install dependencies
go mod download

# Build the generator
go build -o website-generator

# Run the built binary
./website-generator -modules ../modules -output ../website
```

### Command Line Flags

- `-modules` - Path to the modules directory (default: `../modules`)
- `-output` - Path to the output directory (default: `../website`)
- `-serve` - Start a dev server with live reload instead of a one-shot build
- `-port` - Dev server port (default: `8000`, only used with `-serve`)

### Examples

```bash
# Generate to default output directory
go run .

# Generate to custom location
go run . -output /path/to/output

# Use custom modules directory
go run . -modules /path/to/modules -output /path/to/output

# Dev server with live reload on port 3001
go run . -serve -port 3001
```

## How It Works

1. **Reads Markdown Files**: For each configured language, reads one file per module from the modules directory
2. **Converts to HTML**: Uses [goldmark](https://github.com/yuin/goldmark) (CommonMark + GitHub tables) to convert markdown to HTML
3. **Applies Templates**: Wraps content in HTML templates with navigation and styling
4. **Fixes Links**: Converts relative markdown links to HTML links
5. **Generates Index**: Creates an index page with all modules listed
6. **Copies CSS**: Writes the shared stylesheet at the site root

## Languages

Each language is a `LangConfig` in `main.go`:

| Language | File suffix | Output directory |
|---|---|---|
| English | `.md` | `/` |
| Español | `.es.md` | `/es/` |

A `LangConfig` carries the module metadata (titles and descriptions) and the `UIStrings`
for every piece of chrome on the index page. Adding a language means adding one
`LangConfig` and appending it to `languages` — the switcher, output directories and
relative link maths all follow from that.

## Project Structure

```
website-generator/
├── main.go          # Language configs, page generation, HTML templates
├── templates.go     # CSS template
├── serve.go         # Dev server with file watching and live reload
├── go.mod           # Go module definition
└── README.md        # This file
```

## Dependencies

- [goldmark](https://github.com/yuin/goldmark) - Markdown processor

## Generated Output

The generator creates:

- `index.html` - Homepage with the module overview
- `00-setup-and-sanity-check.html` through `09-production-realities.html` - Module pages
- `es/` - The same set of pages in Spanish
- `style.css` - Shared stylesheet

## Customization

### Module Metadata

Edit the `Metadata` list on each `LangConfig` in `main.go` to customize:

- Module titles
- Descriptions
- Filenames

Keep the lists the same length and in the same order — the index, the prev/next
navigation and the language switcher all index into them positionally.

### Templates

Modify the templates:

- `moduleTemplate` in `main.go` - Individual module page layout
- `indexTemplate` in `main.go` - Homepage layout
- `cssTemplate` in `templates.go` - Styling

### Resources and Videos

The "Resources" list and the (optional) embedded talks are per-language data on
`UIStrings`: `ResourceLinks` and `Videos`. Leave `Videos` empty and the whole video
section disappears from the index page.

### Markdown Processing

The `markdownToHTML()` function can be customized to add:

- Custom markdown extensions
- Post-processing steps
- Link transformations

## Regenerating the Website

After making changes to the markdown files:

```bash
make website     # from the repository root
```

Or, while writing, run `make serve` and leave a browser open — the dev server rebuilds
on every save to a `.md` file or to the generator's own sources, and reloads the page.

## Development

### Testing

```bash
# Generate and test locally
go run . -output /tmp/test-website
open /tmp/test-website/index.html
```

### Adding New Modules

1. Add the markdown file to `../modules/` in both languages (`.md`, `.es.md`)
2. Add the metadata entry to every `LangConfig` in `main.go`
3. Run the generator
4. Verify the output

## License

Part of the "Instrumenting your application with OpenTelemetry" workshop project.
