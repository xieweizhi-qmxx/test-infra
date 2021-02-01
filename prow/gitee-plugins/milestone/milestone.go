package milestone

import (
	sdk "gitee.com/openeuler/go-gitee/gitee"
	log "github.com/sirupsen/logrus"
	"regexp"

	prowConfig "k8s.io/test-infra/prow/config"
	plugins "k8s.io/test-infra/prow/gitee-plugins"
	"k8s.io/test-infra/prow/pluginhelp"
)

const (
	unsetMilestoneLabel = "unset-milestone"
	unsetMilestoneComment = "@%s You have not selected a milestone,please select a milestone"
)

var checkMilestoneRe = regexp.MustCompile(`(?mi)^/check-milestone\s*$`)

type milestoneClient interface {
	CreateGiteeIssueComment(org, repo string, number string, comment string) error
}

type milestone struct {
	getPluginConfig plugins.GetPluginConfig
	ghc             *milestoneClient
}

func (m *milestone) HelpProvider(_ []prowConfig.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: "The milestone plugin manages the application and removal of the milestone labels on issue. ",
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/check-milestone",
		Description: "Mandatory check whether the issue is set with milestone,remove or add unset-milestone label",
		Featured:    true,
		WhoCanUse:   "Anyone",
		Examples:    []string{"/check-milestone"},
	})
	return pluginHelp, nil
}

func (m *milestone) PluginName() string {
	return "milestone"
}

func (m *milestone) NewPluginConfig() plugins.PluginConfig {
	return nil
}

func (m *milestone) RegisterEventHandler(p plugins.Plugins) {
	name := m.PluginName()
	p.RegisterIssueHandler(name, m.handleIssueEvent)
	p.RegisterNoteEventHandler(name, m.handleNoteEvent)
}

func (m *milestone) handleIssueEvent(e *sdk.IssueEvent, log *log.Entry) error {
	act := *(e.Action)
	if act == "open" {
		return m.handleIssueCreate(e, log)
	}

	if act == "update" {
		return m.handleIssueUpdate(e, log)
	}
	return nil
}

func (m *milestone) handleNoteEvent(e *sdk.NoteEvent, log *log.Entry) error {
	if *(e.Action) != "comment" {
		log.Debug("Event is not a creation of a comment, skipping.")
		return nil
	}

	if *(e.NoteableType) != "Issue" {
		return nil
	}
	// Only consider "/check-milestone" comments.
	if !checkMilestoneRe.MatchString(e.Comment.Body) {
		return nil
	}
	if e.Issue.Milestone != nil && e.Issue.Milestone.Id != "" {
		if judgeUnMilestoneLabel(e.Issue.Labels) {
			//remove unset-milestone

		} else {
			return nil
		}
	}
	//add unset-milestone
	//add comment
	return nil
}

func (m *milestone) handleIssueCreate(e *sdk.IssueEvent, entry *log.Entry) error {
	return nil
}

func (m *milestone) handleIssueUpdate(e *sdk.IssueEvent, entry *log.Entry) error {
	return nil
}

func judgeUnMilestoneLabel(labs []sdk.LabelHook) bool {
	if len(labs) == 0 {
		return false
	}
	for _, lab := range labs {
		if lab.Name == unsetMilestoneLabel {
			return true
		}
	}
	return false
}
