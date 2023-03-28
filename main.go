package main

import (
	"collector/modules/cpu"
	"collector/modules/disk"
	"collector/modules/ipmi"
	"collector/modules/memory"
	_ "embed"
)

//go:embed bin/*

func main() {

	disk.GetInfo()
	cpu.GetInfo()
	memory.GetInfo()
	ipmi.GetInfo()
}
