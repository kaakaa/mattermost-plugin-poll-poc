package model

import (
	"fmt"

	"github.com/mattermost/mattermost-server/model"
)

type PollOption struct {
	ID string `json:"id"`
	Text string `json:"text"`
}

func (op *PollOption)toPostAction(siteURL, pollId string) *model.PostAction {
	// TODO: fix url
	url := fmt.Sprintf("%s/plugins/matterpoll/polls/%s/answers/%s/vote", siteURL, pollId, op.ID)
	return &model.PostAction{
		Name: op.Text,
		Integration: &model.PostActionIntegration{
			URL: url,
			Context: map[string]interface{}{},
		},
	}
}
