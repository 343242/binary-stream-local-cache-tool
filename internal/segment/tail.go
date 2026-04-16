package segment

import (
	"os"

	"fastReadFile/internal/codec"
	cache "fastReadFile/internal/core"
)

type TailRepairResult struct {
	Size     int64
	Repaired bool
}

func RecoverFooterWithTail(path string, data []byte) (Footer, bool) {
	if len(data) < FooterSize {
		return Footer{}, false
	}
	for start := len(data) - FooterSize; start >= 0; start-- {
		footer, err := DecodeFooter(data[start : start+FooterSize])
		if err != nil {
			continue
		}
		if err := ValidateFooter(path, int64(start+FooterSize), footer); err != nil {
			continue
		}
		return footer, true
	}
	return Footer{}, false
}

func ValidDataEnd(data []byte) (int, error) {
	offset := 0
	for offset < len(data) {
		blockLen, err := codec.BlockLength(data[offset:])
		if err != nil {
			return offset, nil
		}
		if _, err := codec.ParseBlock(data[offset : offset+blockLen]); err != nil {
			return offset, nil
		}
		offset += blockLen
	}
	return offset, nil
}

func RepairTail(path string) (TailRepairResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return TailRepairResult{}, cache.NewError(cache.ErrIO, "repair_segment_tail", path, "read segment", err)
	}
	if footer, ok := RecoverFooterWithTail(path, data); ok {
		size := int64(footer.DataEndOffset) + FooterSize
		repaired, err := TruncateIfLarger(path, size)
		if err != nil {
			return TailRepairResult{}, err
		}
		return TailRepairResult{Size: size, Repaired: repaired}, nil
	}

	validEnd, err := ValidDataEnd(data)
	if err != nil {
		return TailRepairResult{}, err
	}
	repaired, err := TruncateIfLarger(path, int64(validEnd))
	if err != nil {
		return TailRepairResult{}, err
	}
	return TailRepairResult{Size: int64(validEnd), Repaired: repaired}, nil
}

func TruncateIfLarger(path string, size int64) (bool, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return false, cache.NewError(cache.ErrIO, "repair_segment_tail", path, "stat segment", err)
	}
	if stat.Size() <= size {
		return false, nil
	}
	if err := os.Truncate(path, size); err != nil {
		return false, cache.NewError(cache.ErrIO, "repair_segment_tail", path, "truncate segment", err)
	}
	return true, nil
}
