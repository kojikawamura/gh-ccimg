# gh-ccimg

A GitHub CLI extension that extracts all images from GitHub issues and pull requests, with optional direct integration to Claude Code for AI-powered analysis.

## Features

- 🖼️ Extract PNG, JPEG, GIF, and WebP images from GitHub issues and PRs
- 🔒 Secure with built-in size limits and path validation
- ⚡ Fast concurrent downloads (5 parallel by default)
- 💾 Flexible storage: in-memory (base64) or save to disk
- 🤖 Direct Claude Code integration for AI analysis
- 🔑 Uses your existing `gh auth` credentials

## Installation

### Prerequisites
- [GitHub CLI](https://cli.github.com/) installed and authenticated
- Go 1.21+ (for building from source)

### Install from Source
```bash
git clone https://github.com/kojikawamura/gh-ccimg.git
cd gh-ccimg
go build -o gh-ccimg
```

### Install as GitHub CLI Extension
```bash
# After building
gh extension install /path/to/gh-ccimg
```

## Quick Start

### Basic Usage
```bash
# Extract images from an issue
gh-ccimg owner/repo#123

# Extract from a PR using URL
gh-ccimg https://github.com/owner/repo/pull/456
```

### Save to Disk
```bash
# Save images to a directory
gh-ccimg owner/repo#123 --out ./screenshots

# With custom limits
gh-ccimg owner/repo#123 --out ./images --max-size 50 --timeout 30
```

### Claude Integration
```bash
# Send images directly to Claude for analysis
gh-ccimg owner/repo#123 --send "What UI issues do you see in these screenshots?"

# Continue a Claude session
gh-ccimg owner/repo#123 --send "Analyze the design patterns" --continue
```

## Command Reference

### Basic Command
```
gh-ccimg <target> [flags]
```

### Target Formats
- `owner/repo#123` - Issue or PR number
- `https://github.com/owner/repo/issues/123` - Full issue URL
- `https://github.com/owner/repo/pull/456` - Full PR URL

### Flags
| Flag | Description | Default |
|------|-------------|---------|
| `--out`, `-o` | Output directory for images | Memory mode (base64) |
| `--send` | Send images to Claude with prompt | - |
| `--continue` | Continue previous Claude session | false |
| `--max-size` | Maximum image size in MB | 20 |
| `--timeout` | Download timeout in seconds | 15 |
| `--force` | Overwrite existing files | false |

## Usage Examples

### Extract and Analyze Screenshots
```bash
# Perfect for UI/UX review
gh-ccimg design-team/app#789 --send "Review these UI mockups for accessibility issues"
```

### Bug Report Analysis
```bash
# Download error screenshots for debugging
gh-ccimg bugs/critical#101 --out ./bug-screenshots --max-size 10
```

### Documentation Images
```bash
# Extract tutorial images
gh-ccimg docs/wiki#42 --out ./tutorial-images --force
```

### Performance Analysis
```bash
# Analyze performance charts with Claude
gh-ccimg performance/metrics#55 --send "What performance bottlenecks do these charts show?"
```

## Output Formats

### Memory Mode (Default)
When no `--out` directory is specified, images are base64-encoded and printed to stdout:
```
Image 1: data:image/png;base64,iVBORw0KGgoAAAANSUhE...
Image 2: data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEASA...
```

### Disk Mode
When `--out` is specified, images are saved with numbered filenames:
```
./screenshots/
├── img-01.png
├── img-02.jpg
└── img-03.gif
```

## Security Features

- **Path Traversal Protection**: Validates all file paths
- **Content Type Validation**: Only downloads image/* MIME types
- **Resource Limits**: Configurable size and timeout limits
- **File Protection**: Requires `--force` to overwrite existing files
- **No Shell Injection**: Uses secure command execution
- **Auth Delegation**: Leverages `gh` CLI authentication

## Performance

- **Small batches** (≤10 images): Complete in ≤2s + network latency
- **Large batches** (50 images): ≤10s with parallel downloads
- **Memory efficient**: Streaming processing for large files
- **Concurrent**: 5 parallel downloads by default

## Troubleshooting

### Common Issues

**"gh: command not found"**
- Install and authenticate GitHub CLI first

**"Permission denied"**
- Run `gh auth status` to check authentication
- Re-authenticate with `gh auth login` if needed

**"Image too large"**
- Increase `--max-size` limit or filter large images

**"Timeout downloading"**
- Increase `--timeout` or check network connectivity

**"Claude command failed"**
- Ensure Claude CLI is installed and accessible
- Check Claude authentication status

### Getting Help
```bash
gh-ccimg --help
```

## Development

See [CLAUDE.md](CLAUDE.md) for development guidelines and build instructions.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass: `go test ./...`
6. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Related Projects

- [GitHub CLI](https://cli.github.com/) - Official GitHub command line tool
- [Claude Code](https://claude.ai/code) - AI-powered code analysis platform