package syncer

import "gopkg.in/yaml.v3"

type SyncRun struct {
	// The root config file
	RootConfig *RootConfig
	// The subsection of the config file that is relevant to this run
	RunConfig RunConfig
	// The registries of this run
	Registry Registry
	// Where we want to copy destination files to
	DestinationWorkingDir string
}

type RunConfig struct {
	yaml.Node
}

func (r *RunConfig) Decode(v interface{}) error {
	return r.Node.Decode(v)
}
