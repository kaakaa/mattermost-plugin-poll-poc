package model

import (
	"errors"
	"fmt"
	"io"
	"time"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/mattermost/mattermost-server/model"
)

type Poll struct {
	ID string `json:"poll_id"`
	Text string `json:"poll_text"`
	CreatedAt int64 `json:"created_at"`
	Options []PollOption `json:"options`
}

func NewPollFromCommand(args *model.CommandArgs) (*Poll, *model.AppError) {
	poll, err := parseCommandText(args.Command)
	if err != nil {
		return nil, model.NewAppError("NewPollFromCommand", "", nil, err.Error(), 0)
	}

	poll.CreatedAt = time.Now().UnixNano()
	return poll, nil
}

func parseCommandText(text string) (*Poll, error) {
	o := strings.TrimRight(strings.TrimLeft(strings.TrimSpace(strings.TrimPrefix(text, "/matterpoll")), "\""), "\"")
 	if o == "" {
		 return nil, errors.New("Text Parse Error: " + text)
 	}
	options := strings.Split(o, "\" \"")
	if len(options) < 2 {
		return nil, errors.New("Invalid Command Args")
	}
	var op []PollOption
	for i, v := range options[1:] {
		op = append(op, PollOption{
			ID: strconv.Itoa(i),
			Text: v,
		})
	} 
	return &Poll{
		Text: options[0],
		Options: op,
	}, nil
}

func PollFromJson(body io.Reader) (*Poll, *model.AppError) {
	decoder := json.NewDecoder(body)
	var poll Poll
	if err := decoder.Decode(&poll); err != nil {
		return nil, model.NewAppError("PollFromJson", "", nil, err.Error(), 0)
	}
	return &poll, nil
}

func (p *Poll)ToCommandResponseJson(siteURL string) (*model.CommandResponse) {
	var actions []*model.PostAction
	for _, op := range p.Options {
		actions = append(actions, op.toPostAction(siteURL, p.ID))
	}
	actions = append(actions, p.toEndPollAction(siteURL))

	response := &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
		Attachments: []*model.SlackAttachment{
			&model.SlackAttachment{
				Text: p.Text,
				AuthorName: "matterpoll",
				Actions: actions,
			},
		},
	}
	return response
}

func (p *Poll)toEndPollAction(siteURL string) *model.PostAction {
	return &model.PostAction{
		Name: "End Poll",
		Integration: &model.PostActionIntegration {
			// TODO: fix URL
			URL: fmt.Sprintf("%s/plugins/matterpoll/polls/%s/end", siteURL, p.ID),
		},
	}
}

func (p *Poll) ToJson() []byte {
	b, err := json.Marshal(p)
	if err != nil {
		return []byte("")
	} else {
		return b
	}
}