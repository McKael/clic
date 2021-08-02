// Copyright © 2021 Mikael Berthe <mikael@lilotux.net>
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package main

import (
	"os/exec"
)

func execCommand(cmd string, args []string) ([]byte, error) {
	return exec.Command(cmd, args...).Output()
}
