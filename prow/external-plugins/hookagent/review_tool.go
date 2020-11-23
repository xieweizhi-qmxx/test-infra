package main

import (
	"fmt"
	"gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"
	"regexp"
)

const ReviewToolScript = "review_tool"

var (
	RegReview       = regexp.MustCompile(`(?mi)^/review\s*`)
	RegReviewStatus = regexp.MustCompile(`status\[(.*?)\]`)
	RegReviewNumber = regexp.MustCompile(`number_list\[(.*?)\]`)
)

type reviewTool struct {
	process  string
	endpoint string
	name     string
	l        *logrus.Entry
}

func newReviewTool(sc ScriptCfg, log *logrus.Entry) Scenario {
	rt := reviewTool{process: sc.Process, endpoint: sc.Endpoint, name: sc.Name}
	if log != nil {
		log.WithField("script", ReviewToolScript)
	} else {
		log = logrus.StandardLogger().WithField("script", ReviewToolScript)
	}
	rt.l = log
	return &rt
}

func (rt *reviewTool) HandlePullRequestHook(event gitee.PullRequestEvent) {
	if *event.Action == PRUpdate && *event.ActionDesc == PRSourceBranchChange {
		err := rt.handleReviewPullRequest(event.URL)
		if err != nil {
			rt.l.Error(err)
		}
		return
	}

	if *event.Action == PROpen {
		err := rt.handleReviewPullRequest(event.URL)
		if err != nil {
			rt.l.Error(err)
		}
	}

}

func (rt *reviewTool) handleReviewPullRequest(url *string) error {
	if url == nil {
		return fmt.Errorf("the pull request URL is nil")
	}
	cmd, err := ExecCmd(rt.process, rt.endpoint, "-u", *url)
	rt.l.Info(string(cmd))
	if err != nil {
		return err
	}
	//rt.l.Info(string(cmd))
	return nil
}

//HandlePushHook  issue push event processing.
func (rt *reviewTool) HandlePushHook(event gitee.PushEvent) {

}

//HandleNoteHook  comment hook event processing.
func (rt *reviewTool) HandleNoteHook(event gitee.NoteEvent) {
	//action == comment //noteableType = PullRequest //url or pullrequest.htmlUrl //note 为评论内容
	if *event.Action == "comment" && *event.NoteableType == "PullRequest" {
		url := event.PullRequest.HtmlUrl
		comment := *event.Note
		if url == "" || comment == "" {
			return
		}
		err := rt.handlePrReviewComment(comment, url)
		rt.l.Error(err)
	}
}

func (rt *reviewTool) handlePrReviewComment(comment, url string) error {
	ok, status, number := parseCommentToParam(comment)
	if !ok {
		return fmt.Errorf("comments do not need to be processed ")
	}
	params := make([]string, 0, 4)
	params = append(params, rt.endpoint, "-u", url)
	if status != "" {
		params = append(params, "-s", status)
	}
	if number != "" {
		params = append(params, "-e", number)
	}
	cmd, err := ExecCmd(rt.process, params...)
	if err != nil {
		return err
	}
	rt.l.Info(string(cmd))
	return nil
}

//HandleIssueHook issue hook event processing.
func (rt *reviewTool) HandleIssueHook(event gitee.IssueEvent) {

}

func parseCommentToParam(comment string) (bool, string, string) {
	status := ""
	number := ""
	if RegReview.MatchString(comment) {
		if RegReview.MatchString(comment) {
			submatch := RegReviewStatus.FindAllStringSubmatch(comment, 1)
			if len(submatch) > 0 && len(submatch[0]) > 1 {
				status = submatch[0][1]
			}
			submatch = RegReviewNumber.FindAllStringSubmatch(comment, 1)
			if len(submatch) > 0 && len(submatch[0]) > 1 {
				number = submatch[0][1]
			}
			return true, status, number
		}
	}
	return false, "", ""
}
