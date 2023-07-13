package main

import (
	_ "github.com/cresta/syncer/sharedapi/drift/staticfile"
	"github.com/cresta/syncer/sharedapi/syncer"
)

// TODO: Create this file on demand when running the `syncer` CLI rather than force it to already exist
func main() {
	syncer.Sync(syncer.DefaultFxOptions())
}
