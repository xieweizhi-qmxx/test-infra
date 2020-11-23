package main

import (
	"fmt"
	"gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"
)

const ReviewToolScript = "review_tool"

type reviewTool struct {
	process string
	endpoint string
	name string
	l *logrus.Entry
}

func NewReviewTool(proc,ep,name string,log *logrus.Entry) Scenario {
	rt := reviewTool{process:proc,endpoint: ep,name: name}
	if log != nil {
		log.WithField("script", ReviewToolScript)
	}else {
		log = logrus.StandardLogger().WithField("script", ReviewToolScript)
	}
	rt.l = log
	return &rt
}

func (rt *reviewTool) HandlePullRequestHook(event gitee.PullRequestEvent)   {
	if *event.Action == "update" && *event.ActionDesc == "source_branch_changed" {
		err := rt.handleReviewPullRequest(event.URL)
		if err != nil {
			rt.l.Error(err)
		}
	}

}

func (rt *reviewTool) handleReviewPullRequest(url *string) error{
	if url == nil {
		return fmt.Errorf("the pull request URL is nil")
	}
	cmd, err := ExecCmd(rt.process,rt.endpoint,"-u", *url)
        rt.l.Info(string(cmd))
	if err != nil {
		return err
	}
	//rt.l.Info(string(cmd))
	return nil
}

//HandlePushHook  issue push event processing.
func (rt *reviewTool) HandlePushHook(event gitee.PushEvent)  {

}

//HandleNoteHook  comment hook event processing.
func (rt *reviewTool) HandleNoteHook(event gitee.NoteEvent)  {

}

//HandleIssueHook issue hook event processing.
func (rt *reviewTool) HandleIssueHook(event gitee.IssueEvent)  {

}

