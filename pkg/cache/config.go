package cache

import "fastReadFile/internal/core"

type Config = core.Config

func DefaultConfig(rootDir string) Config {
	return core.DefaultConfig(rootDir)
}
