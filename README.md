# podtrawl

A small podcast downloader written in Go.
It reads a list of RSS feeds from a config file and downloads the episode enclosures it finds.
The intention is to be able to automate downloading podcast episodes using cron or similar.

Podtrawl is in early development.
Behaviour and configuration are likely to change.

## Status

Currently downloads every episode of every configured feed into a directory named after the show.
Episodes that have already been downloaded are skipped on subsequent runs.
Directory and episode names are sanitised so that they are safe and portable across common file systems.

## Building

Requires Go 1.27+ and [mage](https://magefile.org).

```sh
mage build   # build into bin/
mage test    # run tests
mage fmt     # run goimports
mage clean   # remove bin/
```

Building with `go build` directly is discouraged, as the mage targets are likely to grow in complexity.

## Configuration

podtrawl looks for `config.toml` in the following locations, in order:

1. `<user config dir>/podtrawl/` (e.g. `%AppData%\podtrawl\` on Windows, `~/.config/podtrawl/` on Linux, `~/Library/Application Support/podtrawl/` on macOS).
2. Next to the executable.

The first file that loads successfully is the config that is used; there is currently no configuration inheritance.

The file lists one `[[feed]]` table per podcast:

```toml
[[feed]]
url = "https://example.com/podcast.rss"

[[feed]]
url = "https://example.org/another/feed.xml"
```

## Running

```sh
./bin/podtrawl
```

Episodes are written to a directory named after the show title, alongside the executable.

## Download cache

podtrawl records what it has already downloaded in `cache.json`,
so that a scheduled run only fetches episodes it has not seen before.
The file lives in `<user cache dir>/podtrawl/`
(e.g. `%LocalAppData%\podtrawl\` on Windows, `~/.cache/podtrawl/` on Linux, `~/Library/Caches/podtrawl/` on macOS).

Episodes are identified by their feed's URL together with the item's "`guid`",
falling back to the enclosure URL for the feeds that omit it.

Deleting the file makes the next run download every episode again.
An unreadable cache is reported on standard error,
treated as empty, and replaced by the next successful download.

## Exit status

podtrawl exits 0 when every feed and episode was handled successfully, and 1 otherwise.
Errors fetching feeds, downloading episodes, or updating the cache are reported on standard error,
but do not stop processing.
If such errors have occured in a run, podtrawl exits 1.

## Planned work

* [x] Track which episodes have been downloaded and don't redownload them
* [x] Sanitize directory and file names
* [ ] Specify the download directory pattern globally, with per-feed overrides
* [ ] Specify the episode filename pattern globally, with per-feed overrides
* [ ] Support podcasting namespaces, such as the `itunes` and `podcast` namespaces
* [ ] Cache feed files and responses (`etag`, `ttl` etc), and respect feed properties (`ttl`, `skipDays`, `skipHours` etc)
* [ ] (Optionally) parallelize downloads across multiple threads
