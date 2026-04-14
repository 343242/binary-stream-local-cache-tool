package fsutil

import (
	"os"
	"sync"

	cache "fastReadFile/internal/core"
)

type TestState struct {
	ReadOnly bool
	DiskFull bool
}

var (
	testStatesMu sync.Mutex
	testStates   = make(map[string]TestState)
)

func SetTestState(root string, state TestState) {
	testStatesMu.Lock()
	defer testStatesMu.Unlock()
	testStates[root] = state
}

func ClearTestState(root string) {
	testStatesMu.Lock()
	defer testStatesMu.Unlock()
	delete(testStates, root)
}

func CheckWritable(root string) error {
	testStatesMu.Lock()
	state, ok := testStates[root]
	testStatesMu.Unlock()
	if ok {
		switch {
		case state.ReadOnly:
			return cache.NewError(cache.ErrReadOnlyFS, "check_writable", root, "filesystem is read-only", nil)
		case state.DiskFull:
			return cache.NewError(cache.ErrDiskFull, "check_writable", root, "disk is full", nil)
		}
	}

	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return cache.NewError(cache.ErrIO, "check_writable", root, "stat root directory", err)
	}
	if info.Mode().Perm()&0o200 == 0 {
		return cache.NewError(cache.ErrReadOnlyFS, "check_writable", root, "root directory is not writable", nil)
	}
	return nil
}
