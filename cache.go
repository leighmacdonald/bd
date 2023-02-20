package main

import (
	"github.com/pkg/errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	errCacheExpired = errors.New("cached value expired")
)

type localCache interface {
	Set(ct cacheType, key string, value io.Reader) error
	Get(ct cacheType, key string, receiver io.Writer) error
}

type fsCache struct {
	rootPath string
	maxAge   time.Duration
}
type cacheType int

const (
	cacheTypeAvatar cacheType = iota
	cacheTypeProfile
	cacheTypeBans
)

func newFsCache(rootDir string, maxAge time.Duration) fsCache {
	cache := fsCache{rootPath: rootDir, maxAge: maxAge}
	cache.init()
	return cache
}

func (cache fsCache) init() {
	for _, p := range []cacheType{cacheTypeAvatar, cacheTypeProfile, cacheTypeBans} {
		if errMkDir := os.MkdirAll(cache.getPath(p, ""), 0770); errMkDir != nil {
			log.Panicf("Failed to setup cache dirs: %v\n", errMkDir)
		}
	}
}

func (cache fsCache) getPath(ct cacheType, key string) string {
	switch ct {
	case cacheTypeAvatar:
		return filepath.Join(cache.rootPath, "avatars", key)
	case cacheTypeBans:
		return filepath.Join(cache.rootPath, "steam_bans", key)
	case cacheTypeProfile:
		return filepath.Join(cache.rootPath, "steam_profile", key)
	default:
		log.Panicf("Got unknown cacheType: %v\n", ct)
		return ""
	}
}

func (cache fsCache) Set(ct cacheType, key string, value io.Reader) error {
	of, errOf := os.OpenFile(cache.getPath(ct, key), os.O_WRONLY|os.O_CREATE, 0660)
	if errOf != nil {
		return errOf
	}
	defer logClose(of)
	if _, errWrite := io.Copy(of, value); errWrite != nil {
		return errWrite
	}
	return nil
}

func (cache fsCache) Get(ct cacheType, key string, receiver io.Writer) error {
	of, errOf := os.Open(cache.getPath(ct, key))
	if errOf != nil {
		return errCacheExpired
	}
	defer logClose(of)

	stat, errStat := of.Stat()
	if errStat != nil {
		return errCacheExpired
	}
	if time.Since(stat.ModTime()) > cache.maxAge {
		return errCacheExpired
	}
	_, errCopy := io.Copy(receiver, of)
	return errCopy
}
