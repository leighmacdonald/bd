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

func (cache FsCache) getPath(ct Type, key string) string {
	switch ct {
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
		cache.logger.Error("Got unknown cache type", "type", ct)
		return ""
	}
}

func (cache FsCache) Set(ct Type, key string, value io.Reader) error {
	fullPath := cache.getPath(ct, key)
	if errMkdir := os.MkdirAll(filepath.Dir(fullPath), os.ModePerm); errMkdir != nil {
		return errMkdir
	}
	of, errOf := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE, 0o660)
	if errOf != nil {
		return errOf
	}
	defer util.LogClose(cache.logger, of)
	if _, errWrite := io.Copy(of, value); errWrite != nil {
		return errWrite
	}
	return nil
}

func (cache FsCache) Get(ct Type, key string, receiver io.Writer) error {
	of, errOf := os.Open(cache.getPath(ct, key))
	if errOf != nil {
		return ErrCacheExpired
	}
	defer util.LogClose(cache.logger, of)

	stat, errStat := of.Stat()
	if errStat != nil {
		return ErrCacheExpired
	}
	if time.Since(stat.ModTime()) > cache.maxAge {
		return ErrCacheExpired
	}
	_, errCopy := io.Copy(receiver, of)
	return errCopy
}
