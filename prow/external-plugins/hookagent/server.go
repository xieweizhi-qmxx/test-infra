package main

import (
	"encoding/json"
	"fmt"
	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/git/v2"
	"k8s.io/test-infra/prow/gitee"
	"k8s.io/test-infra/prow/pluginhelp"
	"net/http"
	"sync"
)


type giteeClient interface {
}

type server struct {
	tokenGenerator func() []byte
	config func() hookAgentConfig
	gec    giteeClient
	gegc   git.ClientFactory
	log    *logrus.Entry
	robot  string
	wg     sync.WaitGroup

}
//GracefulShutdown Handle remaining requests
func (s *server) GracefulShutdown() {
	s.wg.Wait()
}

func helpProvider(enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: "The hookAgent plugin is used to distribute webhook events to third-party scripts.",
	}

	return pluginHelp, nil
}

func (s *server) ServeHTTP(w http.ResponseWriter,r *http.Request) {
	eventType, eventGUID, payload, ok, _ := gitee.ValidateWebhook(w, r, s.tokenGenerator)
	if !ok {
		return
	}
	if err := s.handleEvent(eventType,eventGUID,payload); err != nil {
		s.log.WithError(err)
	}
}

func (s *server) handleEvent(eventType,eventGUID string,payload []byte) error {
	fullName := ""
	var event interface{}
	switch eventType {
	case NoteHookType:
		var e sdk.NoteEvent
		if err := json.Unmarshal(payload, &e); err != nil {
			return err
		}
		fullName = e.Repository.FullName
		event = e
	case IssueHookType:
		var ie sdk.IssueEvent
		if err := json.Unmarshal(payload, &ie); err != nil {
			return err
		}
		fullName = ie.Repository.FullName
		event = ie
	case MergeRequestHookType:
		var pr sdk.PullRequestEvent
		if err := json.Unmarshal(payload, &pr); err != nil {
			return err
		}
		fullName = pr.Repository.FullName
		event = pr
	case PushHookType:
		var pe sdk.PushEvent
		if err := json.Unmarshal(payload, &pe); err != nil {
			return err
		}
		fullName = pe.Repository.FullName
		event = pe
	}
	if fullName == ""{
		return fmt.Errorf("invalidate webhook")
	}
	cfg := s.config()
	scripts := GenScript(cfg.getNeedHandleScript(fullName),s.log)
	if len(scripts) == 0 {
		s.log.Info("No script needs to execute these events")
		return nil
	}
	for _,v := range scripts {
		s.wg.Add(1)
		go func(scenario Script) {
			defer s.wg.Done()
			scenario.ExecScript(eventType,event)
		}(v)
	}
	return nil
}



