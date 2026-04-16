package lock

import (
	"encoding/json"
	"os"
)

func EncodeMetadata(meta Metadata) ([]byte, error) {
	return json.Marshal(meta)
}

func DecodeMetadata(data []byte) (Metadata, error) {
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return Metadata{}, err
	}
	return meta, nil
}

func writeMetadata(file *os.File, meta Metadata) error {
	data, err := EncodeMetadata(meta)
	if err != nil {
		return err
	}
	if err := file.Truncate(0); err != nil {
		return err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	return file.Sync()
}
