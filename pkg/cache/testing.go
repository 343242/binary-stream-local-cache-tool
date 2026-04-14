package cache

var writeBatchHook func(string)

func SetWriteBatchHookForTesting(hook func(string)) {
	writeBatchHook = hook
}
