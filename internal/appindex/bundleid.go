package appindex

import (
	"fmt"
	"os"
	"path/filepath"

	"howett.net/plist"
)

func ReadAppInfo(appPath string) (*AppInfo, error) {
	plistPath := filepath.Join(appPath, "Contents", "Info.plist")
	f, err := os.Open(plistPath)
	if err != nil {
		return nil, fmt.Errorf("open Info.plist: %w", err)
	}
	defer f.Close()

	var info AppInfo
	decoder := plist.NewDecoder(f)
	if err := decoder.Decode(&info); err != nil {
		return nil, fmt.Errorf("decode Info.plist: %w", err)
	}
	return &info, nil
}

func ExtractBundleID(appPath string) (string, error) {
	info, err := ReadAppInfo(appPath)
	if err != nil {
		return "", err
	}
	return info.BundleID, nil
}
