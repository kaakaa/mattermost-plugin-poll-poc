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

func NewPollFromCommand(args *model.CommandArgs) (*Poll, error) {
	poll, err := parseCommandText(args.Command)
	poll.CreatedAt = time.Now().UnixNano()

	if err != nil {
		return nil, err
	}
	return poll, nil
}

func parseCommandText(text string) (*Poll, error) {
	// TODO: not implemented yet
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

func PollFromJson(body io.Reader) (*Poll, error) {
	decoder := json.NewDecoder(body)
	var poll Poll
	if err := decoder.Decode(&poll); err != nil {
		return nil, err
	}
	return &poll, nil
}

func (p *Poll)ToCommandResponseJson() (*model.CommandResponse, error) {
	var actions []*model.PostAction
	for _, op := range p.Options {
		actions = append(actions, op.toPostAction(p.ID))
	}
	actions = append(actions, p.toEndPollAction())

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
	return response, nil
}

func (p *Poll)toEndPollAction() *model.PostAction {
	return &model.PostAction{
		Name: p.Text,
		Integration: &model.PostActionIntegration {
			// TODO: fix URL
			URL: fmt.Sprintf("http://localhost:8065//plugins/matterpoll/polls/%s/end", p.ID),
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