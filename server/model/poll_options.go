package model

import (
	"fmt"

	"github.com/mattermost/mattermost-server/model"
)

type PollOption struct {
	ID string `json:"id"`
	Text string `json:"text"`
}

func (op *PollOption)toPostAction(pollId string) *model.PostAction {
	// TODO: fix url
	url := fmt.Sprintf("http://localhost:8065/plugins/matterpoll/polls/%s/answers/%s/vote", pollId, op.ID)
	return &model.PostAction{
		Name: op.Text,
		Integration: &model.PostActionIntegration{
			URL: url,
		},
	}
}
