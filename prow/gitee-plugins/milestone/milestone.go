package milestone

import (
	"errors"
	"fmt"
	"regexp"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	log "github.com/sirupsen/logrus"

	prowConfig "k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/gitee"
	plugins "k8s.io/test-infra/prow/gitee-plugins"
	"k8s.io/test-infra/prow/pluginhelp"
)

const (
	unsetMilestoneLabel   = "unset-milestone"
	unsetMilestoneComment = "@%s You have not selected a milestone,please select a milestone.After setting the milestone, you can use the /check-milestone command to remove the unset-milestone label."
)

var checkMilestoneRe = regexp.MustCompile(`(?mi)^/check-milestone\s*$`)

type milestoneClient interface {
	CreateGiteeIssueComment(org, repo string, number string, comment string) error
	RemoveIssueLabel(org, repo, number, label string) error
	AddIssueLabel(org, repo, number, label string) error
}

type milestone struct {
	getPluginConfig plugins.GetPluginConfig
	ghc             milestoneClient
}

//NewMilestone create a milestone plugin by config and gitee client
func NewMilestone(f plugins.GetPluginConfig, gec gitee.Client) plugins.Plugin {
	return &milestone{
		getPluginConfig: f,
		ghc:             gec,
	}
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
	if e == nil {
		return errors.New("event payload is nil")
	}
	act := *(e.Action)
	if act == "open" {
		return m.handleIssueCreate(e, log)
	}
	return m.handleIssueUpdate(e)
}

func (m *milestone) handleNoteEvent(e *sdk.NoteEvent, log *log.Entry) error {

	if e == nil {
		return errors.New("event payload is nil")
	}

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
	owner := e.Repository.Namespace
	repo := e.Repository.Path
	number := e.Issue.Number
	author := e.Issue.User.Login
	if e.Issue.Milestone != nil && e.Issue.Milestone.Id != 0 {
		if judgeUnMilestoneLabel(e.Issue.Labels) {
			//remove unset-milestone
			return m.ghc.RemoveIssueLabel(owner, repo, number, unsetMilestoneLabel)
		} else {
			return nil
		}
	}
	return handleAddLabelAndComment(m.ghc, owner, repo, number, author)
}

func (m *milestone) handleIssueCreate(e *sdk.IssueEvent, log *log.Entry) error {
	if e.Milestone != nil && e.Milestone.Id != 0 {
		log.Debug(fmt.Sprintf("Milestones have been set when the issue (%s)was created", e.Issue.Number))
		return nil
	}
	owner := e.Repository.Namespace
	repo := e.Repository.Path
	number := e.Issue.Number
	author := e.Issue.User.Login
	return handleAddLabelAndComment(m.ghc, owner, repo, number, author)
}

func (m *milestone) handleIssueUpdate(e *sdk.IssueEvent) error {
	if e.Issue.Milestone != nil && e.Issue.Milestone.Id != 0 {
		if judgeUnMilestoneLabel(e.Issue.Labels) {
			return m.ghc.RemoveIssueLabel(e.Repository.Namespace, e.Repository.Path, e.Issue.Number, unsetMilestoneLabel)
		}
	}
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

func handleAddLabelAndComment(ghc milestoneClient, owner, repo, number, author string) error {
	err := ghc.AddIssueLabel(owner, repo, number, unsetMilestoneLabel)
	if err != nil {
		return err
	}
	return ghc.CreateGiteeIssueComment(owner, repo, number, fmt.Sprintf(unsetMilestoneComment, author))
}
