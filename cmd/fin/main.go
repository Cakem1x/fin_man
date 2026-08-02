package main

import (
	"log"

	"github.com/Cakem1x/fin_man/cmd/fin/cmd"
)

func main() {
	log.SetFlags(0) // Remove timestamps from log output for a cleaner CLI UX
	cmd.Execute()
}
