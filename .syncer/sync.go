package main

import (
	_ "github.com/cresta/syncer/sharedapi/drift/staticfile"
	"github.com/cresta/syncer/sharedapi/syncer"
)

func main() {
	syncer.Sync()
}
