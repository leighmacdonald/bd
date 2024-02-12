package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

var errCacheExpired = errors.New("cached value expired")

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

// NewCache creates a new local storage backed cache for avatars and player lists.
func NewCache(rootDir string, maxAge time.Duration) (FsCache, error) {
	cache := FsCache{rootPath: rootDir, maxAge: maxAge, logger: slog.Default().WithGroup("cache")}
	if errInit := cache.init(); errInit != nil {
		return FsCache{}, errInit
	}

	return cache, nil
}

// init creates the directory structure used to store locally cached files.
func (cache FsCache) init() error {
	for _, p := range []Type{TypeAvatar, TypeLists} {
		if errMkDir := os.MkdirAll(cache.getPath(p, ""), 0o770); errMkDir != nil {
			return errors.Join(errMkDir, errCacheSetup)
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
		cache.logger.Error("Got unknown cache type", slog.Int("type", int(cacheType)))

		return ""
	}
}

func (cache FsCache) Set(ct Type, key string, value io.Reader) error {
	fullPath := cache.getPath(ct, key)
	if errMkdir := os.MkdirAll(filepath.Dir(fullPath), os.ModePerm); errMkdir != nil {
		return errors.Join(errMkdir, errCreateCacheDir)
	}

	openFile, errOf := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE, 0o660)
	if errOf != nil {
		return errors.Join(errOf, errOpenCacheFile)
	}

	defer LogClose(openFile)

	if _, errWrite := io.Copy(openFile, value); errWrite != nil {
		return errors.Join(errWrite, errWriteCacheFile)
	}

	return nil
}

func (cache FsCache) Get(ct Type, key string, receiver io.Writer) error {
	openFile, errOf := os.Open(cache.getPath(ct, key))
	if errOf != nil {
		return errCacheExpired
	}

	defer LogClose(openFile)

	stat, errStat := openFile.Stat()
	if errStat != nil {
		return errCacheExpired
	}

	if time.Since(stat.ModTime()) > cache.maxAge {
		return errCacheExpired
	}

	_, errCopy := io.Copy(receiver, openFile)
	if errCopy != nil {
		return errors.Join(errCopy, errReadCacheFile)
	}

	return nil
}
