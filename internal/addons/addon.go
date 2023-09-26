package addons

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/pkg/errors"
)

//go:embed *
var addonFS embed.FS

const (
	chatWrapperName = "aaaaaaaaaa_loadfirst_tf2_bot_detector"
	addonNameEraser = "aaaaaaaaaa_votefailed_eraser_v2"
)

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
	if errWalk := fs.WalkDir(src, srcPath, func(path string, fsDir fs.DirEntry, err error) error {
		if fsDir.IsDir() {
			if errMkdir := os.MkdirAll(filepath.Join(dst, path), 0o775); errMkdir != nil {
				return errors.Wrap(errMkdir, "Failed to make output dir")
			}

			return nil
		}

		// Just in case we miss something
		if strings.HasSuffix(fsDir.Name(), ".go") {
			return nil
		}

		data, errData := fs.ReadFile(src, path)
		if errData != nil {
			return errors.Wrap(errData, "Failed to read embed file path")
		}

		outputFile, outputFileErr := os.Create(filepath.Join(dst, path))
		if outputFileErr != nil {
			return errors.Wrap(outputFileErr, "Failed to open output dir")
		}

		if _, errCopy := outputFile.Write(data); errCopy != nil {
			return errors.Wrap(errCopy, "")
		}

		return nil
	}); errWalk != nil {
		return errors.Wrap(errWalk, "Failed to install addon")
	}

	return nil
}
