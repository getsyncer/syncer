package syncer

type SyncRun struct {
	// The root config file
	RootConfig Config
	// The subsection of the config file that is relevant to this run
	RunConfig Config
	// The registries of this run
	Registry Registry
	// Where we want to copy destination files to
	DestinationWorkingDir string
}

type Config struct{}

func (c *Config) UnmarshalInto(v interface{}) error {
	return nil
}
