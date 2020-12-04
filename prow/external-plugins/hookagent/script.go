package main

import (
	"gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"
	"os/exec"
)

const (
	NoteHookType         = "Note Hook"
	IssueHookType        = "IssueHook"
	MergeRequestHookType = "Merge Request Hook"
	PushHookType         = "Push Hook"
	PRUpdate             = "update"
	PROpen               = "open"
	PRSourceBranchChange = "source_branch_changed"
)

//Scenario Interface for handling hook event
type Scenario interface {
	HandleNoteHook(event gitee.NoteEvent)
	HandleIssueHook(event gitee.IssueEvent)
	HandlePullRequestHook(event gitee.PullRequestEvent)
	HandlePushHook(event gitee.PushEvent)
}

//Script Scenario's abstract structure
//all third-party scripts should inherit
type Script struct {
	Scenario
}

func newScript(s Scenario) Script {
	return Script{s}
}

//ExecScript Script processing hook event
func (t *Script) ExecScript(eventType string, event interface{}) {
	switch eventType {
	case NoteHookType:
		if value, ok := event.(gitee.NoteEvent); ok {
			t.HandleNoteHook(value)
		}
	case IssueHookType:
		if value, ok := event.(gitee.IssueEvent); ok {
			t.HandleIssueHook(value)
		}
	case MergeRequestHookType:
		if value, ok := event.(gitee.PullRequestEvent); ok {
			t.HandlePullRequestHook(value)
		}
	case PushHookType:
		if value, ok := event.(gitee.PushEvent); ok {
			t.HandlePushHook(value)
		}
	}

}

//GenScript Generate script based on configuration
func GenScript(scs map[string]ScriptCfg, log *logrus.Entry) []Script {
	var scripts []Script
	for k, v := range scs {
		switch k {
		case ReviewToolScript:
			scripts = append(scripts, newScript(newReviewTool(v, log)))
		}

	}
	return scripts
}

//ExecCmd Command line to execute script
func ExecCmd(ep string, args ...string) ([]byte, error) {
	command := exec.Command(ep, args...)
	return command.Output()
}
