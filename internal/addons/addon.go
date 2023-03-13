package addons

import (
	"embed"
	"fmt"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/pkg/errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed *
var addonFS embed.FS

const chatWrapperName = "aaaaaaaaaa_loadfirst_tf2_bot_detector"
const addonNameEraser = "aaaaaaaaaa_votefailed_eraser_v2"

func Install(tf2dir string) error {
	wrapperPath := filepath.Join(tf2dir, "custom", chatWrapperName)
	if util.Exists(wrapperPath) {
		if errDelete := os.RemoveAll(wrapperPath); errDelete != nil {
			return errors.Wrapf(errDelete, "Failed to remove tf2bd chat wrappers")
		}
	}
	for _, addonName := range []string{addonNameEraser} {
		if errCopy := cpEmbedDir(addonFS, fmt.Sprintf("addons/%s", addonName), tf2dir); errCopy != nil {
			return errors.Wrap(errCopy, "Failed to install votefail eraser addon")
		}
	}
	return nil
}

func cpEmbedDir(src embed.FS, srcPath string, dst string) error {
	return fs.WalkDir(src, srcPath, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			if errMkdir := os.MkdirAll(filepath.Join(dst, path), 0775); errMkdir != nil {
				return errMkdir
			}
		} else {
			// Just in case we miss something
			if strings.HasSuffix(d.Name(), ".go") {
				return nil
			}
			data, errData := fs.ReadFile(src, path)
			if errData != nil {
				return errData
			}
			outputFile, outputFileErr := os.Create(filepath.Join(dst, path))
			if outputFileErr != nil {
				return outputFileErr
			}
			if _, errCopy := outputFile.Write(data); errCopy != nil {
				return errCopy
			}
		}
		return nil
	})
}
