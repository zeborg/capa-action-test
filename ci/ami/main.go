package main

import (
	"flag"
	"log"

	ghaction "github.com/zeborg/capa-action-test/github-action"
	"github.com/zeborg/capa-action-test/prow"
)

func main() {
	mode := flag.String("mode", "", "Acceptable values: 'github' (for CAPA GitHub Action) and 'prow' (for CAPA Prow Jobs)")
	flag.Parse()

	if *mode == "github" {
		ghaction.GH()
	} else if *mode == "presubmit" {
		prow.Presubmit()
	} else if *mode == "postsubmit" {
		prow.Postsubmit()
	} else if *mode == "" {
		log.Fatal("Error: No value provided for 'mode' flag")
	} else {
		log.Fatalf("Error: Invalid value '%s' found for 'mode' flag", *mode)
	}
}
