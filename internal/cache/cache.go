package cache

import (
	"fmt"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/pkg/errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	ErrCacheExpired = errors.New("cached value expired")
)

type Cache interface {
	Set(ct Type, key string, value io.Reader) error
	Get(ct Type, key string, receiver io.Writer) error
}

type FsCache struct {
	rootPath string
	maxAge   time.Duration
}
type Type int

const (
	TypeAvatar Type = iota
	TypeLists
)

func New(rootDir string, maxAge time.Duration) FsCache {
	cache := FsCache{rootPath: rootDir, maxAge: maxAge}
	cache.init()
	return cache
}

func (cache FsCache) init() {
	for _, p := range []Type{TypeAvatar, TypeLists} {
		if errMkDir := os.MkdirAll(cache.getPath(p, ""), 0770); errMkDir != nil {
			log.Panicf("Failed to setup cache dirs: %v\n", errMkDir)
		}
	}
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
		log.Panicf("Got unknown Type: %v\n", ct)
		return ""
	}
}

func (cache FsCache) Set(ct Type, key string, value io.Reader) error {
	fullPath := cache.getPath(ct, key)
	if errMkdir := os.MkdirAll(filepath.Dir(fullPath), os.ModePerm); errMkdir != nil {
		return errMkdir
	}
	of, errOf := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE, 0660)
	if errOf != nil {
		return errOf
	}
	defer util.LogClose(of)
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
	defer util.LogClose(of)

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
