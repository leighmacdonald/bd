package addons

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"errors"
)

//go:embed *
var addonFS embed.FS

var (
	ErrRemoveWrappers  = errors.New("failed to remove tf2bd chat wrappers")
	ErrInstallVoteFail = errors.New("Failed to install votefail eraser addon")
	ErrCreateOutput    = errors.New("Failed to make output dir")
	ErrReadEmbed       = errors.New("Failed to read embed file path")
	ErrOpenOutput      = errors.New("Failed to open output dir")
	ErrWriteOutput     = errors.New("Failed to write output file")
	ErrInstall         = errors.New("failed to install addon")
)

const (
	chatWrapperName = "aaaaaaaaaa_loadfirst_tf2_bot_detector"
	addonNameEraser = "aaaaaaaaaa_votefailed_eraser_v2"
)

func Install(tf2dir string) error {
	wrapperPath := filepath.Join(tf2dir, "custom", chatWrapperName)
	if exists(wrapperPath) {
		if errDelete := os.RemoveAll(wrapperPath); errDelete != nil {
			return errors.Join(errDelete, ErrRemoveWrappers)
		}
	}

	for _, addonName := range []string{addonNameEraser} {
		if errCopy := cpEmbedDir(addonFS, fmt.Sprintf("addons/%s", addonName), tf2dir); errCopy != nil {
			return errors.Join(errCopy, ErrInstallVoteFail)
		}
	}

	return nil
}

func cpEmbedDir(src embed.FS, srcPath string, dst string) error {
	if errWalk := fs.WalkDir(src, srcPath, func(path string, fsDir fs.DirEntry, err error) error {
		if fsDir.IsDir() {
			if errMkdir := os.MkdirAll(filepath.Join(dst, path), 0o775); errMkdir != nil {
				return errors.Join(errMkdir, ErrCreateOutput)
			}

			return nil
		}

		// Just in case we miss something
		if strings.HasSuffix(fsDir.Name(), ".go") {
			return nil
		}

		data, errData := fs.ReadFile(src, path)
		if errData != nil {
			return errors.Join(errData, ErrReadEmbed)
		}

		outputFile, outputFileErr := os.Create(filepath.Join(dst, path))
		if outputFileErr != nil {
			return errors.Join(outputFileErr, ErrOpenOutput)
		}

		if _, errCopy := outputFile.Write(data); errCopy != nil {
			return errors.Join(errCopy, ErrWriteOutput)
		}

		return nil
	}); errWalk != nil {
		return errors.Join(errWalk, ErrInstall)
	}

	return nil
}

func exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	return true
}
