package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/plugin/rpcplugin"
	poll_model "github.com/matterpoll/matterpoll/server/model"
	"github.com/matterpoll/matterpoll/server/store"
)

var (
	voteRoute = regexp.MustCompile(`/polls/([0-9a-z]+)/answers/([0-9]+)/vote`)
	endPollRoute = regexp.MustCompile(`/polls/([0-9a-z]+)/end`)
)

type MatterPollPlugin struct{
	api plugin.API
	configuration atomic.Value
	store *store.MatterPollStore
	hogeru string
}

func (p *MatterPollPlugin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch  {
	case voteRoute.MatchString(r.URL.Path):
		p.handleVote(w, r)
	case endPollRoute.MatchString(r.URL.Path):
		p.handleEndPoll(w, r)
	default:
		fmt.Fprint(w, "FUFUFFUFUFUF")
	}
}



func (p MatterPollPlugin) ExecuteCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	poll, err := poll_model.NewPollFromCommand(args)
	// TODO: fix error messages
	if err != nil {
		return nil, model.NewAppError("here", "NewPollFromCommand", map[string]interface{}{}, err.Error(), 1)
	}
	if poll == nil {
		return nil, model.NewAppError("here", "NilPoll", map[string]interface{}{}, "hoge", 1)
	}
	if err = p.store.CreatePoll(poll); err != nil {
		return nil, model.NewAppError("here", "CreatePoll", map[string]interface{}{}, "hoge", 1)
	}
	resp, err := poll.ToCommandResponseJson()
	if err != nil {
		return nil, model.NewAppError("here", "ToCommandResponse", map[string]interface{}{}, err.Error(), 1)
	}
	log.Printf("%v", resp.ToJson())
	log.Printf("%v", resp.Attachments[0].Text)
	log.Printf("%v", resp.Attachments[0].Actions[0].Integration)
	log.Printf("%v", resp.Attachments[0].Actions[0].Integration.URL)
	log.Printf("%v", resp.Attachments[0].Actions[0].Integration.Context)
	return resp, nil

}

func (p *MatterPollPlugin) handleVote(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	var integrationReq model.PostActionIntegrationRequest
	if err := decoder.Decode(&integrationReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ret := voteRoute.FindAllStringSubmatch(r.URL.Path, 1)
	pollId := ret[0][1]
	voteId := ret[0][2]
	updated, err := p.store.Vote(pollId, voteId, integrationReq.UserId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Vote Error: %s", err)
		return		
	}

	var resp *model.PostActionIntegrationResponse
	if updated {
		resp = &model.PostActionIntegrationResponse{
			EphemeralText: "Success to change an answer",
		}
	} else {
		resp = &model.PostActionIntegrationResponse{
			EphemeralText: "Success to vote an answer",
		}
	}
	b, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Vote Error: %s", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func (p *MatterPollPlugin) handleEndPoll(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	var integrationReq model.PostActionIntegrationRequest
	if err := decoder.Decode(&integrationReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	matches := endPollRoute.FindAllStringSubmatch(r.URL.Path, 1)
	pollId := matches[0][1]

	answers, err := p.store.ReadPollAnswers(pollId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Vote Error: %s", err)
		return
	}
	var summary map[string][]string
	for user, answer := range answers {
		if v, ok := summary[answer]; ok {
			summary[answer] = append(v, user)
		} else {
			summary[answer] = []string{user}
		}
	}

	poll, err := p.store.ReadPoll(pollId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Vote Error: %s", err)
		return
	}

	var ret []string
	for _, op := range poll.Options {
		if v, ok := answers[op.ID]; ok {
			ret = append(ret, fmt.Sprintf("%s: %s: %v", op.ID, op.Text, v))
		} else {
			ret = append(ret, fmt.Sprintf("%s: %s: []", op.ID, op.Text))
		}
	}
	resp := &model.PostActionIntegrationResponse {
		Update: &model.Post {
			Message: strings.Join(ret, "\n"),
		},
	}
	b, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Vote Error: %s", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}


func (p *MatterPollPlugin) OnActivate(api plugin.API) error {
	p.api = api
	p.store = store.NewMatterPollStore(api.KeyValueStore())
	/*
	if err := p.OnConfigurationChange(); err != nil {
		return err
	}
	*/

	/*
	config := p.configuration.Load().(*Configuration)
	if err := config.IsValid(); err != nil {
		return err
	}

	p.hogeru = config.Hogeru
	_, err := p.api.GetTeamByName("tttt")
	if err != nil {
		return err
	}
	*/
	command := &model.Command{
		Id: "hoge_plugin_command",
		// TeamId: t.Id,
		Trigger: "matterpoll",
		// Method: "POST",
		// URL: "/plugins/hoge1/hello",
		AutoComplete: true,
		AutoCompleteDesc: "hohohohogehgoehogheoghoehoge",
		AutoCompleteHint: "ohogehogehogehoge",
	}
	if err := p.api.RegisterCommand(command); err != nil {
		return err
	}
	return nil
}

/*
func (p *MatterPollPlugin)OnConfigurationChange() error {
	var configuration interface{}
	err := p.api.LoadPluginConfiguration(&configuration)
	p.configuration.Store(&configuration)
	return err
}
*/
func main() {
	rpcplugin.Main(&MatterPollPlugin{})
}