//go:build mage

package main

import "github.com/magefile/mage/sh"

const binDir = "bin/"

func Build() error {
	return sh.RunV("go", "build", "-o", binDir, ".")
}

func Fmt() error {
	return sh.RunV("go", "tool", "goimports", "-format-only", "-w", "-l", ".")
}

func Test() error {
	return sh.RunV("go", "test", "./...")
}

func Clean() error {
	return sh.Rm(binDir)
}
