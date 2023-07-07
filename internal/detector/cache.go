package detector

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/pkg/errors"
	"golang.org/x/exp/slog"
)

var ErrCacheExpired = errors.New("cached value expired")

type Cache interface {
	Set(ct Type, key string, value io.Reader) error
	Get(ct Type, key string, receiver io.Writer) error
}

type NopCache struct{}

func (c *NopCache) Set(_ Type, _ string, _ io.Reader) error {
	return nil
}

func (c *NopCache) Get(_ Type, _ string, _ io.Writer) error {
	return nil
}

type FsCache struct {
	rootPath string
	maxAge   time.Duration
	logger   *slog.Logger
}

type Type int

const (
	TypeAvatar Type = iota
	TypeLists
)

func NewCache(logger *slog.Logger, rootDir string, maxAge time.Duration) (FsCache, error) {
	cache := FsCache{rootPath: rootDir, maxAge: maxAge, logger: logger.WithGroup("cache")}
	if errInit := cache.init(); errInit != nil {
		return FsCache{}, errInit
	}

	return cache, nil
}

func (cache FsCache) init() error {
	for _, p := range []Type{TypeAvatar, TypeLists} {
		if errMkDir := os.MkdirAll(cache.getPath(p, ""), 0o770); errMkDir != nil {
			return errors.Wrap(errMkDir, "Failed to setup cache dirs")
		}
	}

	return nil
}

func (cache FsCache) getPath(cacheType Type, key string) string {
	switch cacheType {
	case TypeAvatar:
		if key == "" {
			return filepath.Join(cache.rootPath, "avatars")
		}

		prefix := key[0:2]
		root := filepath.Join(cache.rootPath, "avatars", prefix)

		return filepath.Join(root, fmt.Sprintf("%s.jpg", key))
	case TypeLists:
		return filepath.Join(cache.rootPath, "lists", key)
	default:
		cache.logger.Error("Got unknown cache type", "type", cacheType)

		return ""
	}
}

func (cache FsCache) Set(ct Type, key string, value io.Reader) error {
	fullPath := cache.getPath(ct, key)
	if errMkdir := os.MkdirAll(filepath.Dir(fullPath), os.ModePerm); errMkdir != nil {
		return errors.Wrap(errMkdir, "Failed to make output path")
	}

	openFile, errOf := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE, 0o660)
	if errOf != nil {
		return errors.Wrap(errOf, "Failed to open output file")
	}

	defer util.LogClose(cache.logger, openFile)

	if _, errWrite := io.Copy(openFile, value); errWrite != nil {
		return errors.Wrap(errWrite, "Failed to write output file")
	}

	return nil
}

func (cache FsCache) Get(ct Type, key string, receiver io.Writer) error {
	openFile, errOf := os.Open(cache.getPath(ct, key))
	if errOf != nil {
		return ErrCacheExpired
	}

	defer util.LogClose(cache.logger, openFile)

	stat, errStat := openFile.Stat()
	if errStat != nil {
		return ErrCacheExpired
	}

	if time.Since(stat.ModTime()) > cache.maxAge {
		return ErrCacheExpired
	}

	_, errCopy := io.Copy(receiver, openFile)
	if errCopy != nil {
		return errors.Wrap(errCopy, "Failed to copy to output file")
	}

	return nil
}
