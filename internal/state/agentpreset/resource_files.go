package agentpreset

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"reasonix/internal/base/secrets"
	"reasonix/internal/contract/agentcomposition"
)

func CaptureFiles(paths []string) (DirectorySnapshot, map[string]string, error) {
	asset := DirectorySnapshot{Files: []AssetFile{}}
	mapping := map[string]string{}
	for i, path := range paths {
		info, data, err := readAssetFile(path)
		if err != nil {
			return asset, nil, err
		}
		rel := fmt.Sprintf("files/%d/%s", i, filepath.Base(path))
		mapping[filepath.Clean(path)] = rel
		asset.Files = append(asset.Files, AssetFile{Path: filepath.ToSlash(rel), Data: data, Mode: uint32(info.Mode().Perm() & 0o700)})
		if err := validateAssets(asset); err != nil {
			return asset, nil, err
		}
	}
	return asset, mapping, nil
}

func readAssetFile(path string) (os.FileInfo, []byte, error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, nil, err
	}
	defer root.Close()
	name := filepath.Base(path)
	info, err := root.Lstat(name)
	if err != nil {
		return nil, nil, err
	}
	if !info.Mode().IsRegular() || secrets.SensitiveFileName(info.Name()) || info.Size() > maxAssetBytes {
		return nil, nil, fmt.Errorf("%w: unsupported hook resource", agentcomposition.ErrUnavailable)
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxAssetBytes+1))
	if len(data) > maxAssetBytes {
		return nil, nil, agentcomposition.ErrUnavailable
	}
	return info, data, err
}
