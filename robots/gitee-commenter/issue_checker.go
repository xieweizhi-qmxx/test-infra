package main

import (
	"github.com/sirupsen/logrus"
)

type issueChecker struct {
	log          *logrus.Entry
	client commenterClient
	cItem checkItem

	org string
	repo string
	number string

}

func (pc *issueChecker) check() {
	//Get the details again and ensure that the status is up to date as much as possible
	issue, err := pc.client.GetIssue(pc.org, pc.repo, pc.number)
	if err != nil {
		pc.log.Error(err)
		return
	}
	if !needCheckIssue(issue,pc.cItem){
		return
	}
	err = pc.client.CreateGiteeIssueComment(pc.org, pc.repo, pc.number, pc.cItem.Comment)
    if err != nil {
    	pc.log.Error(err)
	}
}