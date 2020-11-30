package repohandle

import (
	"encoding/base64"
	"fmt"
	"strings"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	prowConfig "k8s.io/test-infra/prow/config"
	plugins "k8s.io/test-infra/prow/gitee-plugins"
	"k8s.io/test-infra/prow/pluginhelp"

)

var cacheFilePath = "RepoHandleCacheConfig.json"

type repoHandle struct {
	rhc          *rhClient
	cache        *cacheProcessedFile
	getPluginCfg plugins.GetPluginConfig
}

func NewRepoHandle(f plugins.GetPluginConfig, gec giteeClient) plugins.Plugin {
	return &repoHandle{
		getPluginCfg: f,
		rhc:          &rhClient{giteeClient: gec},
		cache:        newCache(cacheFilePath),
	}
}

func (rh *repoHandle) PluginName() string {
	return "repoHandle"
}

func (rh *repoHandle) HelpProvider(_ []prowConfig.OrgRepo) (*pluginhelp.PluginHelp, error) {
	return nil, nil
}

func (rh *repoHandle) RegisterEventHandler(p plugins.Plugins) {
	pn := rh.PluginName()
	p.RegisterPushEventHandler(pn, rh.handlePushEvent)
}

func (rh repoHandle) NewPluginConfig() plugins.PluginConfig {
	return &configuration{}
}

//HandleDefaultTask handle this plugin default task, tasks not triggered by webhook
func (rh *repoHandle) HandleDefaultTask() {
	log := logrus.WithFields(
		logrus.Fields{
			"component":  rh.PluginName(),
			"event-type": "defaultTask",
		},
	)
	err := rh.cache.cacheInit()
	if err != nil {
		log.Error(err)
	}
	go rh.handleRepoDT(log)
}

func (rh *repoHandle) handlePushEvent(e *sdk.PushEvent, log *logrus.Entry) error {
	files, err := rh.getNeedHandleFiles()
	if err != nil {
		return err
	}
	if len(files) == 0 || e == nil {
		return nil
	}
	//check the push commit file in the need handle files
	idx, find := getPushEventChangeFile(e, files)
	if !find {
		return nil
	}
	for k := range idx {
		err = rh.handleRepoConfigFile(&files[k], log)
		if err != nil {
			log.Error(err)
		}
	}
	return rh.cache.saveCache(files)
}

func (rh *repoHandle) getPluginConfig() (*configuration, error) {
	cfg := rh.getPluginCfg(rh.PluginName())
	if cfg == nil {
		return nil, fmt.Errorf("can't find the configuration")
	}

	c, ok := cfg.(*configuration)
	if !ok {
		return nil, fmt.Errorf("can't convert to configuration")
	}
	return c, nil
}

func (rh *repoHandle) handleRepoDT(log *logrus.Entry) {
	files, err := rh.getNeedHandleFiles()
	if err != nil {
		log.Error(err)
		return
	}
	if len(files) == 0 {
		return
	}
	for k := range files {
		err = rh.handleRepoConfigFile(&files[k], log)
		if err != nil {
			log.Error(err)
		}
	}
	err = rh.cache.saveCache(files)
	if err != nil {
		log.Error(err)
	}
}

func (rh *repoHandle) handleRepoConfigFile(file *cfgFilePath, log *logrus.Entry) error {
	content, err := rh.rhc.getRealPathContent(file.Owner, file.Repo, file.Path, file.Ref)
	if err != nil {
		return err
	}
	if content.Sha == "" || content.Content == "" || content.Sha == file.Hash {
		log.Info(fmt.Sprintf("%s/%s/%s configuration does not need to be processed", file.Owner, file.Repo, file.Path))
		return nil
	}
	decodeBytes, err := base64.StdEncoding.DecodeString(content.Content)
	if err != nil {
		return err
	}
	rc := Repos{}
	err = yaml.UnmarshalStrict(decodeBytes, &rc)
	if err != nil {
		return err
	}
	if rc.Community == "" || len(rc.Repositories) == 0 {
		return fmt.Errorf("repos configuration error")
	}
	for _, v := range rc.Repositories {
		err = rh.handleAddRepository(rc.Community, v, log)
		if err != nil {
			log.Error(err)
		}
	}
	file.Hash = content.Sha
	return nil
}

func (rh *repoHandle) handleAddRepository(community string, repository Repository, log *logrus.Entry) error {
	repo, ex, err := rh.rhc.existRepo(community, *repository.Name)
	if err != nil {
		return err
	}
	//handle rename repo first
	if !ex && repository.RenameFrom != nil && (*repository.RenameFrom) != "" {
		rnRepo, exist, err := rh.rhc.existRepo(community, *repository.RenameFrom)
		if err != nil {
			return err
		}
		if !exist {
			return fmt.Errorf("repository defined by rename_from does not exist: %s ", *repository.RenameFrom)
		}
		return rh.rhc.updateRepoName(community, rnRepo.Name, *repository.Name)
	}
	if !ex {
		//add repo on gitee
		repo, err = rh.rhc.createRepo(community, *repository.Name, *repository.Description, *repository.Type, repository.AutoInit)
		if err != nil {
			return err
		}
		// add branch on repo
		if len(repository.ProtectedBranches) > 0 {
			for _, v := range repository.ProtectedBranches {
				if v != "" && v != repo.DefaultBranch {
					_, err = rh.rhc.giteeClient.CreateBranch(community, repo.Name, repo.DefaultBranch, v)
					if err != nil {
						log.Error(err)
					}
				}
			}
		}

	}
	//setting branch
	err = rh.handleRepoBranchProtected(community, repository)
	if err != nil {
		log.Error(err)
	}
	//repo setting
	err = rh.handleRepositorySetting(&repo, repository)
	if err != nil {
		return err
	}
	return nil
}

func (rh *repoHandle) handleRepositorySetting(repo *sdk.Project, repository Repository) error {
	if repo == nil {
		return fmt.Errorf("gitee %s repository is empty", *repository.Name)
	}
	owner := repo.Namespace.Path
	if owner == "" {
		return fmt.Errorf("repository %s information not obtained", *repository.Name)
	}
	exceptType := "private" == *repository.Type
	exceptCommentable := repository.IsCommentable()
	typeChange := repo.Private != exceptType
	commentChange := repo.CanComment != exceptCommentable
	if typeChange || commentChange {
		pt := repo.Private
		pc := repo.CanComment
		if typeChange {
			pt = exceptType
		}
		if commentChange {
			pc = exceptCommentable
		}
		err := rh.rhc.updateRepoCommentOrType(owner, *repository.Name, pc, pt)
		return err
	}
	return nil
}

func (rh *repoHandle) handleRepoBranchProtected(community string, repository Repository) error {
	// if the branches are defined in the repositories, it means that
	// all the branches defined in the community will not inherited by repositories
	branch, err := rh.rhc.GetRepoAllBranch(community, *repository.Name)
	if err != nil {
		return err
	}
	cbm := make(map[string]int, len(branch))
	for k, v := range branch {
		cbm[v.Name] = k
	}
	nbm := make(map[string]string, len(repository.ProtectedBranches))
	for _, v := range repository.ProtectedBranches {
		nbm[v] = v
	}
	//remove protected config dose not exist in current branches when branch is protected
	for k, v := range cbm {
		if branch[v].Protected {
			if _, exist := nbm[k]; !exist {
				err = rh.rhc.CancelBranchProtected(community, *repository.Name, k)
				if err == nil {
					branch[v].Protected = false
				}
			}
		}
	}
	//add protected current config branch on repository
	for k := range nbm {
		if v, exist := cbm[k]; exist && branch[v].Protected == false {
			_, err = rh.rhc.SetBranchProtected(community, *repository.Name, k)
			if err != nil {
				logrus.Println(err)
			}
		}
	}
	return nil
}

func (rh *repoHandle) getNeedHandleFiles() ([]cfgFilePath, error) {
	var repoFiles []cfgFilePath
	c, err := rh.getPluginConfig()
	if err != nil {
		return repoFiles, err
	}
	for _, f := range c.RepoHandler.RepoFiles {
		if f.Owner != "" && f.Repo != "" && f.Path != "" {
			repoFiles = append(repoFiles, f)
		}
	}
	cacheConfig, err := rh.cache.loadCache()
	if err != nil {
		return repoFiles, nil
	}
	if len(cacheConfig) > 0 {
		for k := range repoFiles {
			for _, v := range cacheConfig {
				if repoFiles[k].equal(v) {
					repoFiles[k].Hash = v.Hash
				}
			}
		}
	}
	return repoFiles, nil
}

func getPushEventChangeFile(e *sdk.PushEvent, files []cfgFilePath) ([]int, bool) {
	fns := make(map[string]struct{})
	var fIdx []int
	for _, v := range e.Commits {
		if len(v.Added) > 0 {
			for _, fn := range v.Added {
				fns[fn] = struct{}{}
			}
		}
		if len(v.Modified) > 0 {
			for _, fn := range v.Modified {
				fns[fn] = struct{}{}
			}
		}
	}
	if len(fns) == 0 {
		return fIdx, false
	}
	find := false
	for k, v := range files {
		if e.Repository.Namespace != v.Owner || e.Repository.Name != v.Repo || e.Ref == nil {
			continue
		}
		ref := v.Ref
		if ref == "" {
			ref = "master"
		}
		if !strings.Contains(*e.Ref, ref) {
			continue
		}
		if _, ex := fns[v.Path]; ex {
			find = true
			fIdx = append(fIdx, k)
		}
	}
	return fIdx, find
}
