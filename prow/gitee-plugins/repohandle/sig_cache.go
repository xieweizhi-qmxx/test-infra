package repohandle

import "sync"

var (
	sigInstance *sigCache
	sigOnce     sync.Once
)

type sigCache struct {
	sigMap map[string]string
	sync.Mutex
}

func getSigInstance() *sigCache {
	sigOnce.Do(func() {
		sigInstance = &sigCache{}
		sigInstance.sigMap = make(map[string]string, 0)
	})
	return sigInstance
}

func (sc *sigCache) init(sigList []sig) {
	sc.Lock()
	sc.sigMap = make(map[string]string, 0)
	for _, v := range sigList {
		for _, s := range v.Repositories {
			sc.sigMap[s] = v.Name
		}
	}
	sc.Unlock()
}

//sigChange When the sig configuration file changes,
// call this method to get the repository that needs to be processed
func (sc *sigCache) sigChange(sigList []sig) []string {
	var cRepo []string
	if len(sigList) == 0 {
		return cRepo
	}
	sc.Lock()
	for _, v := range sigList {
		for _, s := range v.Repositories {
			sg, ok := sc.sigMap[s]
			if !ok {
				sc.sigMap[s] = v.Name
				cRepo = append(cRepo, s)
			} else {
				if sg != v.Name {
					sc.sigMap[s] = v.Name
					cRepo = append(cRepo, s)
				}
			}
		}
	}
	sc.Unlock()
	return cRepo
}

func (sc *sigCache) loadSigName(repo string) string {
	sc.Lock()
	sn := sc.sigMap[repo]
	sc.Unlock()
	return sn
}

//ownerChane When the owner configuration file changes,
// call this method to get the repository that needs to be processed
func (sc *sigCache) ownerChane(sn string) []string {
	var cRepo []string
	if sn == "" {
		return cRepo
	}
	sc.Lock()
	for k, v := range sc.sigMap {
		if sn == v {
			cRepo = append(cRepo, k)
		}
	}
	sc.Unlock()
	return cRepo
}


