# podtrawl

A small podcast downloader written in Go.
It reads a list of RSS feeds from a config file and downloads the episode enclosures it finds.
The intention is to be easy to work with from the command line,
while also supporting scheduled downloads using cron or similar.

Podtrawl is in early development.
Behaviour and configuration are likely to change.

## Status

Currently downloads every episode of every configured feed to a user-configurable location.
Episodes that have already been downloaded are skipped on subsequent runs.
Names are sanitised so that they are safe and portable across common file systems.

## Building

Requires Go 1.27+ and [mage](https://magefile.org).

```sh
mage build  # build into bin/
mage test   # run tests
mage fmt    # run goimports
mage clean  # remove bin/
```

Building with `go build` directly is discouraged, as the mage targets are likely to grow in complexity.

## Configuration

podtrawl looks for `config.toml` in the following locations, in order:

1. `<user config dir>/podtrawl/` (e.g. `%AppData%\podtrawl\` on Windows, `~/.config/podtrawl/` on Linux, `~/Library/Application Support/podtrawl/` on macOS).
2. Next to the executable.

The first file that loads successfully is the config that is used; there is currently no configuration inheritance.

Passing `-c, --config <path>` uses that file instead, and the locations above are not searched.

The file lists one `[[feed]]` table per podcast.
The optional `[defaults]` table specifies behaviour to be shared between feeds.

```toml
# Root-level options come first, before any table.
root = "D:/Podcasts"

# Shared by every feed that does not override them.
# Options left out here, such as dirname, keep their built-in defaults.
[defaults]
# Use a literal string to avoid having to escape the double quotes in the template.
filename = '{{ date "2006-01-02" .Episode.PubDate }} {{ sanitize .Episode.Title }}{{ .Episode.Ext }}'
existing = "skip"

# Takes all of its settings from [defaults].
[[feed]]
url = "https://example.com/podcast.rss"

# Overrides two of them.
[[feed]]
url = "https://example.org/another/feed.xml"
dirname = 'Archive/{{ sanitize .Feed.Title }}'
filename = '{{ .Episode.Filename }}'
```

Under this config, if the first feed had an episode titled "Episode 12: Naming Things"
and published on the 16th of August 2026, it would be written to
`D:/Podcasts/<show title>/2026-08-16 Episode 12- Naming Things.mp3`.

### Root-level options

The following options must be set before any tables in the config file.

#### `root` (string, optional)

Specifies the directory under which downloads are saved.
Defaults to `<user downloads dir>/podtrawl/`
(e.g. `%UserProfile%\Downloads\podtrawl\` on Windows, `~/Downloads/podtrawl/` on Linux and macOS).
Must be an absolute path.

### `[defaults]` and `[[feed]]` options

The following options may be set in the `[defaults]` table, or in any of the `[[feed]]` tables.
Where an option is absent from a `[[feed]]` table, its value will be taken from the `[defaults]` table.
If unset in the `[defaults]` table, the built-in default will be used.

#### `dirname` (string, optional)

Go template specifying the directory to which a feed's episodes are written, under `root`.
Must be a relative path.
Defaults to `{{ sanitizeOr .Feed.Title (printf "show-%s" (shorthash .Feed.Url)) }}`.

Rendered once per feed, with access to the following metadata:

| Field | Type | Description |
| --- | --- | --- |
| `.Feed.Title` | `string` | The channel's title. |
| `.Feed.Link` | `string` | The show's web page. |
| `.Feed.Description` | `string` | The channel's description. |
| `.Feed.Url` | `string` | The feed's URL, as configured. Will never be the empty string. |

#### `filename` (string, optional)

Go template specifying the file to which an episode is written, under its feed's `dirname`.
Must be a relative path.
Defaults to `{{ sanitize (printf "%s%s" (or (sanitize .Episode.Stem) (printf "episode-%s" (shorthash .Episode.Url))) .Episode.Ext) }}`.

Rendered per episode, with access to the same metadata as `dirname`, plus:

| Field | Type | Description |
| --- | --- | --- |
| `.Episode.Title` | `string` | The item's title. |
| `.Episode.Link` | `string` | The episode's web page. |
| `.Episode.Description` | `string` | The item's description. |
| `.Episode.Guid` | `string` | The item's guid. |
| `.Episode.PubDate` | `time.Time` | The item's publication date. Format with the `date` function. |
| `.Episode.Url` | `string` | The enclosure's URL. |
| `.Episode.Type` | `string` | The enclosure's media type. |
| `.Episode.Length` | `int` | The enclosure's size, in bytes. Frequently absent or wrong. |
| `.Episode.Filename` | `string` | The enclosure's filename, as extracted from its URL. |
| `.Episode.Stem` | `string` | `.Episode.Filename` without the extension. |
| `.Episode.Ext` | `string` | File extension, including the dot. Taken from `Episode.Filename`, falling back to one inferred from `.Episode.Type`. |

#### `sanitize` (boolean, optional)

Whether rendered names are made safe for common file systems.
Defaults to `true`.

If `true`, the following transformations are applied to each path segment after a name is rendered:

* Characters that are illegal on one or more common file systems, including control characters, are replaced with `-`;
* Leading spaces, and trailing spaces and dots are trimmed;
* A stem that is a reserved Windows name, such as `nul`, is suffixed with `_`;
* Segments longer than 255 bytes are truncated, preserving the extension where possible.

Use the [`sanitize` function](#functions) to constrain a value to a single segment.

#### `existing` (enumerated string, optional)

What to do if the destination of an episode to be downloaded refers to an existing file.
Defaults to `increment`.
One of:

| Policy | Behaviour |
| --- | --- |
| `increment` | Append a monotonically increasing integer to the filename's stem (e.g. `episode.mp3` -> `episode (1).mp3`), and leave the existing file alone. Bails after 1000 attempts. |
| `skip` | Do not download the episode, and leave the existing file alone. |
| `replace` | Download the episode, replacing the original file. |

### `[[feed]]`-only options

The following options may only be set in `[[feed]]` tables.

#### `url` (string, required)

URL to the podcast's RSS feed.
Every `[[feed]]` table must specify a `url`.
Each `url` must be unique within the config.

### Functions

In addition to those provided by `text/template`,
`dirname` and `filename` may also use the following functions:

| Function | Description |
| --- | --- |
| `sanitize s` | Make `s` safe as a single path component, replacing path separators and applying the same rules as the `sanitize` option. Gives the empty string if nothing is left. |
| `sanitizeOr s fallback` | As `sanitize`, but uses `fallback` when `s` yields nothing. If used, `fallback` is sanitized too. If `fallback` is also unusable, gives `_`. |
| `shorthash s` | A short, stable, name-safe digest of `s`. Uses the non-cryptographic 64-bit FNV-1a hash function. |
| `date layout t` | Format a time using a [Go layout](https://pkg.go.dev/time#pkg-constants), such as `2006-01-02`. |
| `lower s`, `upper s` | Change case. |
| `trim s` | Remove leading and trailing whitespace. |
| `replace old new s` | Replace every `old` in `s` with `new`. |

### Notes

* No variable or tilde expansion is performed for `root`, `dirname` or `filename`.
* Slashes are accepted as path separators on Windows.
* Regardless of the setting of `sanitize`, `dirname` and `filename` must be relative, and must not include upward traversal.

## Running

```sh
./bin/podtrawl [-c <path>] [-n] [-q|-v] [--json]
```

By default, episodes are downloaded to `<user downloads dir>/podtrawl/<feed title>/<enclosure filename>`.
Where no usable feed title or enclosure filename is available, `show-<feed URL hash>` or `episode-<enclosure URL hash>`, respectively, is used.
Use the `root`, `dirname` and `filename` configuration options to customize where downloads are stored.

Log output is written to standard error.
The default verbosity shows errors and warnings.
The output verbosity can be changed with the `-v, --verbose` or `-q, --quiet` flags:

* `-v`: Set log level to debug
* `-q`: Set log level to error
* `-qq`: set log level to above error.

The `--json` flag may be passed to emit logs as line-delimited JSON.

Report output, such as that produced by `--dry-run`, is written to standard output.
It is not affected by the `--verbose` or `--quiet` arguments.

## Dry run

`-n, --dry-run` reports what a run would do, without downloading any episodes or modifying the cache.
Feeds are still fetched.

Every episode is listed with where it would be written, and what would happen to it:

```
download  /home/you/Downloads/podtrawl/Test Podcast/episode2-644.mp3
cached    /home/you/Downloads/podtrawl/Test Podcast/episode1-440.mp3
skip      /home/you/Downloads/podtrawl/Test Podcast/episode0-trailer.mp3
```

* `download`: the episode would be fetched and written to that path.
* `cached`: the cache already accounts for it, so it would not be fetched.
* `skip`: a file is already there and the `existing` option is `skip`.

## Download cache

podtrawl records what it has already downloaded in `cache.json`,
so that a scheduled run only fetches episodes it has not seen before.
The file lives in `<user cache dir>/podtrawl/`
(e.g. `%LocalAppData%\podtrawl\` on Windows, `~/.cache/podtrawl/` on Linux, `~/Library/Caches/podtrawl/` on macOS).

Episodes are identified by their feed's URL together with the item's "`guid`",
falling back to the enclosure URL for the feeds that omit it.

Deleting the file makes the next run download every episode again.
An unreadable cache is logged at level warn,
treated as empty, and replaced by the next successful download.

## Exit status

podtrawl exits 0 when every feed and episode was handled successfully,
2 if there were issues with the arguments passed at the CLI,
and 1 otherwise.
Errors fetching feeds, downloading episodes, or updating the cache are logged,
but do not stop processing.
If such errors have occurred in a run, podtrawl exits 1.

## Planned work

* [x] Track which episodes have been downloaded and don't redownload them
* [x] Sanitise directory and file names
* [x] Specify the download directory pattern globally, with per-feed overrides
* [x] Specify the episode filename pattern globally, with per-feed overrides
* [ ] Support podcasting namespaces, such as the `itunes` and `podcast` namespaces
* [ ] Cache feed files and responses (`etag`, `ttl` etc), and respect feed properties (`ttl`, `skipDays`, `skipHours` etc)
* [ ] (Optionally) parallelize downloads across multiple threads
* [ ] Import and export subscriptions to/from OPML
* [ ] Interactive configuration and cache management
