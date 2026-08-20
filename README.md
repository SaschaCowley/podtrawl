# podtrawl

A small podcast downloader written in Go.
It reads a list of RSS feeds from a config file and downloads the episode enclosures it finds.
The intention is to be able to automate downloading podcast episodes using cron or similar.

Podtrawl is in early development.
Behaviour and configuration are likely to change.

## Status

Currently downloads every episode of every configured feed on each run into a directory named after the show.
Directory and episode names are not sanitised.

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


## Planned work

* [ ] Track which episodes have been downloaded and don't redownload them
* [ ] Sanitize directory and file names
* [ ] Specify the download directory pattern globally, with per-feed overrides
* [ ] Specify the episode filename pattern globally, with per-feed overrides
* [ ] Support podcasting namespaces, such as the `itunes` and `podcast` namespaces
* [ ] Cache feeds (`etag`, `ttl` etc), and respect feed properties (`ttl`, `skipDays`, `skipHours` etc)
* [ ] (Optionally) parallelize downloads across multiple threads
