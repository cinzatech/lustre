# lustre

A local, live-reloading diff viewer for git branches, powered by [difftastic](https://difftastic.wilfred.me.uk/) for structural, syntax-aware diffs.

Point it at a branch and it'll open a side-by-side diff view in your browser, with per-token change highlighting, language detection, and automatic refresh when files change on disk.

Built for small teams that want a lightweight review flow without overhead.

## Prerequisites

- **Go 1.22+**: [go.dev/dl](https://go.dev/dl/)
- **git**
- **difftastic**: install via `cargo install --locked difftastic`, or grab a binary from [releases](https://github.com/Wilfred/difftastic/releases). Make sure `difft` is on your `PATH`.

## Install from source

```
git clone <your-repo-url> lustre
cd lustre
make build
```

Move the binary somewhere on your `PATH`:

```
mv lustre ~/.local/bin/       # or /usr/local/bin/, or wherever you prefer
```

## Development

Building and linting require the following tools:

- **make**: used to drive all build and lint tasks.
- **Go 1.22+**: [go.dev/dl](https://go.dev/dl/)
- **staticcheck**: install with `go install honnef.co/go/tools/cmd/staticcheck@latest`. Used by `make lint` for static analysis.
- **Biome**: installed on demand via `npx @biomejs/biome`. Used by `make fmt` and `make lint` to format and check HTML, CSS, and JS files.

Common targets:

```
make build    # compile the binary to ./lustre
make fmt      # auto-format Go and frontend sources
make lint     # run go vet, staticcheck, and biome check
make check    # fmt + lint
make install  # build and install to $PREFIX/bin (default /usr/local/bin)
make clean    # remove the built binary
```

## Usage

From inside any git repository:

```
lustre <branch> [base]
```

`base` defaults to `main` if omitted.

```
lustre feature/mfa            # diff feature/mfa against main
lustre feature/mfa develop    # diff feature/mfa against develop
```

This starts a local server, opens your default browser, and watches for file changes. Edit code, save, and the diff view updates automatically. `Ctrl+C` to stop.

## License

lustre is shared under the terms of the GNU Affero General Public License Version 3. Find a copy of the full license text in our [LICENSE.md](./LICENSE.md) file.
