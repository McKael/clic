// Copyright © 2021 Mikael Berthe <mikael@lilotux.net>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3"
)

func main() {
	fs := flag.NewFlagSet("cli", flag.ExitOnError)
	var (
		dbPath  = fs.String("db", "clic.sqlite3", "sqlite3 DB path")
		ttl     = fs.Duration("ttl", 120*time.Second, "cache TTL")
		verbose = fs.Bool("verbose", false, "log verbose information")
		init    = fs.Bool("init", false, "create/initialize database")
		getOnly = fs.Bool("get", false, "get cached result")
		clean   = fs.Bool("clean", false, "clean up old entries (wrt TTL)")
		_       = fs.String("config", "", "config file (optional)")
	)

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "CLIC is a CLI tool that adds a cache to commands output.\n\n")
		fmt.Fprintf(fs.Output(), "Usage:\n")
		fmt.Fprintf(fs.Output(), "  clic -h|-help|--help\n")
		fmt.Fprintf(fs.Output(), "  clic [-db database] -init\n")
		fmt.Fprintf(fs.Output(), "  clic [options...] [--] COMMAND [COMMAND_ARGS...]\n")
		fmt.Fprintf(fs.Output(), "\nOptions:\n")
		fs.PrintDefaults()
	}

	ff.Parse(fs, os.Args[1:],
		ff.WithEnvVarPrefix("CLIC"),
		ff.WithConfigFileFlag("config"),
		ff.WithConfigFileParser(ff.PlainParser),
	)

	if *init {
		if *verbose {
			log.Printf("Initializing DB file [%s]...\n", *dbPath)
		}
		if err := dbCreate(*dbPath); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	if !*clean && len(fs.Args()) == 0 {
		return
	}

	if *verbose {
		log.Printf("Using cache DB file [%s]\n", *dbPath)
	}

	var sqlH *sqlHandler
	var err error

	if fileExists(*dbPath) {
		sqlH, err = dbOpen(*dbPath)
		if err != nil {
			log.Printf("Error opening DB file: %v\n", err)
		} else {
			defer sqlH.Close()
		}
	} else {
		log.Println("Error: DB file does not exist (use -init)")
	}

	if *clean {
		if *verbose {
			log.Printf("Cleaning outdated entries (older than %v)...\n", *ttl)
		}
		if err := sqlH.Clean(time.Now().Add(-*ttl).Unix()); err != nil {
			log.Fatal(err)
		}
	}

	if len(fs.Args()) > 0 {
		getOrRun(sqlH, fs.Args(), *ttl, *getOnly, *verbose)
	}
}

func getOrRun(dbh *sqlHandler, cmd []string, ttl time.Duration, getOnly, verbose bool) error {
	cmdString := fmt.Sprintf("%#v", cmd)
	val, timestamp, err := dbh.GetItem(cmdString)
	if err != nil {
		if verbose {
			if strings.Contains(errors.Unwrap(err).Error(), "no rows in result set") {
				log.Printf("No cached result\n")
			} else {
				log.Printf("Result=%#v, ts=%d, err=%v\n", val, timestamp, err)
			}
		}
	} else { // err == nil
		// Is the cached value still valid?
		if time.Now().Sub(time.Unix(timestamp, 0)) <= ttl {
			if verbose {
				log.Printf("Using cached result (timestamp %d)\n", timestamp)
			}
			fmt.Print(val)
			return nil
		}
		if verbose {
			log.Printf("Found expired result in cache (timestamp %d)\n", timestamp)
		}
	}
	if getOnly {
		return nil
	}

	if verbose {
		log.Println("Running external command...")
	}
	valBytes, err := execCommand(cmd[0], cmd[1:])
	if err != nil {
		log.Fatalf("Cannot run command: %v\n", err)
	}
	val = string(valBytes)

	if verbose {
		log.Println("Storing item in cache DB...")
	}
	if err = dbh.SetItem(cmdString, val); err != nil {
		log.Printf("Cannot update cache: %v", err)
	}

	fmt.Print(val)
	return nil
}

func fileExists(filename string) bool {
	if _, err := os.Stat(filename); err != nil {
		return false
	}
	return true
}
