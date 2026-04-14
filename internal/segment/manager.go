package segment

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fastReadFile/internal/codec"
	cache "fastReadFile/internal/core"
	"fastReadFile/internal/wal"
)

type Manager struct {
	cfg             cache.Config
	root            string
	dir             string
	current         *SegmentFile
	tailRepairCount uint64
}

func OpenManager(root string, cfg cache.Config) (*Manager, error) {
	cfg = ensureDefaults(cfg)
	dir := filepath.Join(root, "segments")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, cache.NewError(cache.ErrIO, "open_segment_manager", dir, "create segment directory", err)
	}

	current, tailRepairs, err := openCurrentSegment(root, dir, cfg)
	if err != nil {
		return nil, err
	}

	return &Manager{
		cfg:             cfg,
		root:            root,
		dir:             dir,
		current:         current,
		tailRepairCount: tailRepairs,
	}, nil
}

func (m *Manager) CurrentSegmentID() uint64 {
	if m.current == nil {
		return 0
	}
	return m.current.id
}

func (m *Manager) AppendBlock(block []byte, meta BlockMeta) (AppendResult, error) {
	if m.current.size > 0 && m.current.size+int64(len(block)) > m.cfg.SegmentTargetSizeBytes+m.cfg.SegmentSlackSizeBytes {
		if _, err := m.SealActive(); err != nil {
			return AppendResult{}, err
		}
		if err := m.rotate(); err != nil {
			return AppendResult{}, err
		}
	}
	return m.current.AppendBlock(block, meta)
}

func (m *Manager) SealActive() (Footer, error) {
	if m.current == nil {
		return Footer{}, cache.NewError(cache.ErrIO, "seal_segment_manager", m.dir, "no active segment", nil)
	}
	return m.current.Seal()
}

func (m *Manager) Close() error {
	if m.current == nil {
		return nil
	}
	return m.current.Close()
}

func (m *Manager) SyncActive() error {
	if m.current == nil {
		return nil
	}
	return m.current.Sync()
}

func (m *Manager) TailRepairCount() uint64 {
	return m.tailRepairCount
}

func (m *Manager) rotate() error {
	if err := m.current.Close(); err != nil {
		return err
	}
	nextID := m.current.id + 1
	current, err := openSegmentFile(filepath.Join(m.dir, segmentFileName(nextID)), nextID, m.cfg)
	if err != nil {
		return err
	}
	m.current = current
	return nil
}

type segmentFileInfo struct {
	id     uint64
	path   string
	sealed bool
}

func openCurrentSegment(root, dir string, cfg cache.Config) (*SegmentFile, uint64, error) {
	segments, err := scanSegments(dir)
	if err != nil {
		return nil, 0, err
	}
	if len(segments) == 0 {
		file, err := openSegmentFile(filepath.Join(dir, segmentFileName(1)), 1, cfg)
		return file, 0, err
	}

	latest := segments[len(segments)-1]
	if latest.sealed {
		nextID := latest.id + 1
		file, err := openSegmentFile(filepath.Join(dir, segmentFileName(nextID)), nextID, cfg)
		return file, 0, err
	}

	recovered, err := rebuildActiveSegmentState(root, latest.path)
	if err != nil {
		return nil, 0, err
	}
	file, err := openSegmentFileWithState(latest.path, latest.id, cfg, &recovered)
	return file, recovered.tailRepairs, err
}

func scanSegments(dir string) ([]segmentFileInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, cache.NewError(cache.ErrIO, "scan_segments", dir, "read segment directory", err)
	}

	segments := make([]segmentFileInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".seg") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		id, err := segmentIDFromPath(path)
		if err != nil {
			return nil, cache.NewError(cache.ErrIO, "scan_segments", path, "parse segment id", err)
		}
		_, footerErr := ReadFooter(path)
		segments = append(segments, segmentFileInfo{
			id:     id,
			path:   path,
			sealed: footerErr == nil,
		})
	}
	sort.Slice(segments, func(i, j int) bool { return segments[i].id < segments[j].id })
	return segments, nil
}

func rebuildActiveSegmentState(root, path string) (recoveredSegmentState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return recoveredSegmentState{}, cache.NewError(cache.ErrIO, "rebuild_active_segment", path, "read active segment", err)
	}

	checkpoint, err := wal.NewCheckpointStore(root).Load()
	if err != nil {
		return recoveredSegmentState{}, err
	}
	walBatchSeq, err := walBatchSeqByBlock(root)
	if err != nil {
		return recoveredSegmentState{}, err
	}

	recovered := recoveredSegmentState{lastBatchSeq: checkpoint.LastBatchSeq}
	offset := 0
	for offset < len(data) {
		blockLen, err := codec.BlockLength(data[offset:])
		if err != nil {
			if err := truncateActiveTail(path, int64(offset)); err != nil {
				return recoveredSegmentState{}, err
			}
			recovered.tailRepairs++
			return recovered, nil
		}

		blockBytes := data[offset : offset+blockLen]
		block, err := codec.ParseBlock(blockBytes)
		if err != nil {
			if err := truncateActiveTail(path, int64(offset)); err != nil {
				return recoveredSegmentState{}, err
			}
			recovered.tailRepairs++
			return recovered, nil
		}

		recovered.recordCount += uint64(block.RecordCount)
		if recovered.firstWriteSeq == 0 {
			recovered.firstWriteSeq = block.FirstWriteSeq
			recovered.minEventTime = block.MinEventTime
			recovered.maxEventTime = block.MaxEventTime
		}
		recovered.lastWriteSeq = block.LastWriteSeq
		if block.MinEventTime < recovered.minEventTime {
			recovered.minEventTime = block.MinEventTime
		}
		if block.MaxEventTime > recovered.maxEventTime {
			recovered.maxEventTime = block.MaxEventTime
		}
		if batchSeq, ok := walBatchSeq[string(blockBytes)]; ok {
			recovered.lastBatchSeq = batchSeq
		}

		offset += blockLen
	}
	return recovered, nil
}

func walBatchSeqByBlock(root string) (map[string]uint64, error) {
	log, err := wal.Open(root)
	if err != nil {
		return nil, err
	}
	defer log.Close()

	entries, err := log.Scan()
	if err != nil {
		return nil, err
	}

	byBlock := make(map[string]uint64, len(entries))
	for _, entry := range entries {
		byBlock[string(entry.Block)] = entry.BatchSeq
	}
	return byBlock, nil
}

func truncateActiveTail(path string, size int64) error {
	if err := os.Truncate(path, size); err != nil {
		return cache.NewError(cache.ErrIO, "rebuild_active_segment", path, "truncate corrupt active tail", err)
	}
	return nil
}

func segmentFileName(id uint64) string {
	return strings.TrimSpace(filepath.Base(formatSegmentID(id) + ".seg"))
}

func formatSegmentID(id uint64) string {
	return leftPadUint(id, 6)
}

func leftPadUint(id uint64, width int) string {
	text := []byte{}
	for value := id; value > 0; value /= 10 {
		text = append([]byte{byte('0' + value%10)}, text...)
	}
	if len(text) == 0 {
		text = []byte{'0'}
	}
	for len(text) < width {
		text = append([]byte{'0'}, text...)
	}
	return string(text)
}

func ensureDefaults(cfg cache.Config) cache.Config {
	defaults := cache.DefaultConfig(cfg.RootDir)
	if cfg.RootDir != "" {
		defaults.RootDir = cfg.RootDir
	}
	if cfg.SegmentTargetSizeBytes != 0 {
		defaults.SegmentTargetSizeBytes = cfg.SegmentTargetSizeBytes
	}
	if cfg.SegmentSlackSizeBytes != 0 {
		defaults.SegmentSlackSizeBytes = cfg.SegmentSlackSizeBytes
	}
	if cfg.SegmentFsyncInterval != 0 {
		defaults.SegmentFsyncInterval = cfg.SegmentFsyncInterval
	}
	if cfg.SegmentFsyncBytes != 0 {
		defaults.SegmentFsyncBytes = cfg.SegmentFsyncBytes
	}
	return defaults
}
