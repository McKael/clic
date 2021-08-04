# CLIC — a CLI Caching tool

`clic` is a small program that adds very dumb caching support for command line tools.

It can be useful when the output of a slow command needs to be used several times.
The cache is currently backed by a sqlite3 database.

## Usage

```
% clic --help       # Display usage & options
Usage:
  clic -h|-help|--help
  clic [-db database] -init
  clic [options...] [--] COMMAND [COMMAND_ARGS...]

Options:
  -clean
        clean up old entries (wrt TTL)
  -config string
        config file (optional)
  -db string
        sqlite3 DB path (default "clic.sqlite3")
  -get
        get cached result
  -init
        create/initialize database
  -ttl duration
        cache TTL (default 2m0s)
  -verbose
        log verbose information
```

All options can be provided through an environment variable (e.g. `CLIC_DB`) or a plaintext configuration file (option value).

The database must be initialized (`-init` creates the DB with the cache table, or clears the table if it already exists) before the cache can be used:
```
% export CLIC_DB="$HOME/clic.db"
% clic -init
```

You can use the `-verbose` flag to see what's going on:
```
% clic -verbose -init
2021/08/04 22:50:29 Initializing DB file [/home/mikael/clic.db]...
```

A simple example with `date`:
```
% clic -verbose date +%H:%M:%S
2021/08/04 23:04:41 Using cache DB file [/home/mikael/clic.db]
2021/08/04 23:04:41 No cached result
2021/08/04 23:04:41 Running external command...
2021/08/04 23:04:41 Storing item in cache DB...
23:04:41
% sleep 2; clic -ttl 4s -- date +%H:%M:%S   # cache still valid, not running date
23:04:41
% sleep 2; clic -ttl 4s -- date +%H:%M:%S   # cache entry older than 4s
23:04:45
% sleep 2; clic -ttl 4s -- date +%H:%M:%S   # cache still valid...
23:04:45
```

Reuse a slow Kubernetes cluster query:
```
% clic -- kubectl get pods --all-namespaces -o wide | grep node-7
% clic -- kubectl get pods --all-namespaces -o wide | grep node-8
```

Same, but considering that all entries older than 30 seconds can't be used:
```
% clic -ttl 30s -- kubectl get pods --all-namespaces -o wide | grep node-8
```

Cache a slow find query result:
```
% time clic sudo find /usr -name "*clic*" > /dev/null
clic sudo find /usr -name "*clic*" > /dev/null  0.35s user 0.32s system 3% cpu 18.153 total
% time clic sudo find /usr -name "*clic*" > /dev/null
clic sudo find /usr -name "*clic*" > /dev/null  0.00s user 0.00s system 109% cpu 0.008 total
```

Drop all cache entries older than 1 hour:
```
% clic -clean -ttl 1h
```

## Installation

The usual process for Golang builds should work fine:

    go install github.com/McKael/clic@latest

Note: `clic` uses `mattn/go-sqlite3`, which currently uses cgo.

## Issues

* The standard error of the called process is discarded.

```
% clic ls /lost+found
2021/08/04 23:25:15 Error: command failed: exit status 2
```

A workaround could be (in some cases) to use a shell redirection:
```
% clic bash -c "ls /lost+found 2>&1"
ls: cannot open directory '/lost+found': Permission denied
2021/08/04 23:25:24 Error: command failed: exit status 2
```

* When the command fails, the standard output is still displayed but the cache is not updated (so the command should be run again next time).
