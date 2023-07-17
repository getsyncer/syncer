package files

import (
	"fmt"
	"path/filepath"
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

func (f *System[T]) Paths() []Path {
	paths := make([]Path, 0, len(f.files))
	for path := range f.files {
		paths = append(paths, path)
	}
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
			ret.files[path] = state
		}
	}
	return &ret, nil
}
