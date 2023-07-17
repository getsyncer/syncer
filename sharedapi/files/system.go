package files

import (
	"fmt"
	"path/filepath"
	"sort"

	"go.uber.org/zap/zapcore"
)

type Path string

func (f Path) Clean() Path {
	return Path(filepath.Clean(string(f)))
}

func (f Path) String() string {
	return string(f)
}

type System[T Validatable] struct {
	files map[Path]T
}

type Validatable interface {
	Validate() error
}

func (f *System[T]) Add(path Path, state T) error {
	if f.files == nil {
		f.files = make(map[Path]T)
	}
	path = path.Clean()
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if err := state.Validate(); err != nil {
		return fmt.Errorf("invalid state for %s: %w", path, err)
	}
	if _, ok := f.files[path]; ok {
		return fmt.Errorf("file %s already exists", path)
	}
	f.files[path] = state
	return nil
}

func (f *System[T]) MarshalLogObject(e zapcore.ObjectEncoder) error {
	if err := e.AddArray("paths", zapcore.ArrayMarshalerFunc(func(enc zapcore.ArrayEncoder) error {
		for _, path := range f.Paths() {
			file := f.Get(path)
			if err := enc.AppendObject(zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
				enc.AddString("path", string(path))
				if err := enc.AddReflected("state", file); err != nil {
					return fmt.Errorf("failed to marshal state: %w", err)
				}
				return nil
			})); err != nil {
				return fmt.Errorf("failed to marshal path %s: %w", path, err)
			}
		}
		return nil
	})); err != nil {
		return fmt.Errorf("failed to marshal paths: %w", err)
	}
	return nil
}

var _ zapcore.ObjectMarshaler = &System[Validatable]{}

func (f *System[T]) Paths() []Path {
	paths := make([]Path, 0, len(f.files))
	for path := range f.files {
		paths = append(paths, path)
	}
	sort.SliceStable(paths, func(i, j int) bool {
		return paths[i] < paths[j]
	})
	return paths
}

func (f *System[T]) Get(path Path) T {
	return f.files[path]
}

func (f *System[T]) IsTracked(path Path) bool {
	if f.files == nil {
		return false
	}
	_, ok := f.files[path]
	return ok
}

func (f *System[T]) RemoveTracked(path Path) error {
	if f.files == nil {
		return fmt.Errorf("file %s does not exist", path)
	}
	if _, ok := f.files[path]; !ok {
		return fmt.Errorf("file %s does not exist", path)
	}
	delete(f.files, path)
	return nil
}

type MergeDuplicatePathErr[T Validatable] struct {
	Path   Path
	Value1 T
	Value2 T
}

func (e *MergeDuplicatePathErr[T]) Error() string {
	return fmt.Sprintf("duplicate path %s: %v %v", e.Path, e.Value1, e.Value2)
}

func SystemMerge[T Validatable](systems ...*System[T]) (*System[T], error) {
	var ret System[T]
	for _, system := range systems {
		for path, state := range system.files {
			if ret.IsTracked(path) {
				return nil, &MergeDuplicatePathErr[T]{Path: path, Value1: ret.Get(path), Value2: state}
			}
			if err := ret.Add(path, state); err != nil {
				return nil, fmt.Errorf("failed to add %s: %w", path, err)
			}
		}
	}
	return &ret, nil
}
