package main

import (
	_ "github.com/cresta/syncer/sharedapi/drift/staticfile"
	"github.com/cresta/syncer/sharedapi/syncer"
)

// TODO: Synth this file and run it from the CLI
func main() {
	syncer.Sync(syncer.DefaultFxOptions())
}
