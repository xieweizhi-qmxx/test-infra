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
)

type Scenario interface {
	HandleNoteHook(event gitee.NoteEvent)
	HandleIssueHook(event gitee.IssueEvent)
	HandlePullRequestHook(event gitee.PullRequestEvent)
	HandlePushHook(event gitee.PushEvent)
}

type Script struct {
	Scenario
}

func (t *Script) Exec(eventType string, event interface{}) {
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

func CreateScript(scs map[string]ScriptCfg, log *logrus.Entry) []Script {
	var scripts []Script
	for k, v := range scs {
		switch k {
		case ReviewToolScript:
			sec := NewReviewTool(v.Endpoint, v.Name, log)
			scripts = append(scripts, Script{sec})
		}

	}
	return scripts
}

func ExecCmd(args ...string)( []byte,error) {
	command := exec.Command("python3", args...)
	return command.Output()
}
