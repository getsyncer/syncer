package syncer

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeConfig(t *testing.T) {
	cl := DefaultConfigLoader{
		childrenRegistry: &childrenRegistry{},
	}
	config1 := `
version: 1
config:
  a: b
`
	ctx := context.Background()
	root, err := cl.LoadConfig(ctx, bytes.NewReader([]byte(config1)))
	require.NoError(t, err)
	require.Equal(t, 1, root.Version)
	var configType map[string]string
	require.NoError(t, root.Config.Decode(&configType))
	require.Equal(t, map[string]string{"a": "b"}, configType)
	config2 := `
version: 1
config:
  c: d
`
	root2, err := cl.LoadConfig(ctx, bytes.NewReader([]byte(config2)))
	require.NoError(t, err)
	require.Equal(t, 1, root2.Version)
	var configType2 map[string]string
	require.NoError(t, root2.Config.Decode(&configType2))
	require.Equal(t, map[string]string{"c": "d"}, configType2)
	require.NoError(t, root2.Merge(root))

	var configType3 map[string]string
	require.NoError(t, root2.Config.Decode(&configType3))
	require.Equal(t, map[string]string{"a": "b", "c": "d"}, configType3)
}
