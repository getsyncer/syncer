//go:build syncer
// +build syncer

package main

import (
     _ "github.com/getsyncer/public-sync-modules/opensourcegocli"
	"github.com/getsyncer/syncer-core/syncerexec"
)

func main() {
	syncerexec.FromCli(syncerexec.DefaultFxOptions())
}
