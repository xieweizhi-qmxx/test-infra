package repohandle

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"
)

var (
	instance *cache
	once     sync.Once
)

type cache struct {
	filePath string
	sync.Mutex
}

func newCache(filePath string) *cache {
	once.Do(func() {
		instance = &cache{filePath: filePath}
	})
	return instance
}

func (c *cache) cacheInit() error {
	if exist(c.filePath) {
		return nil
	}
	file, err := os.Create(c.filePath)
	if err != nil {
		return err
	}
	err = file.Close()
	return err
}

func (c *cache) loadCache() ([]repoFile,error) {
	c.Lock()
	defer c.Unlock()
	var cacheRepos []repoFile
	data, err := ioutil.ReadFile(c.filePath)
	if err != nil {
		return cacheRepos,err
	}
	err = json.Unmarshal(data, &cacheRepos)
	return cacheRepos,err
}

func (c *cache) saveCache(repoFiles []repoFile) error {
	data, err := json.Marshal(&repoFiles)
	if err != nil {
		return err
	}
	c.Lock()
	defer c.Unlock()
	file, err := os.OpenFile(c.filePath, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	return err
}

func exist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}
