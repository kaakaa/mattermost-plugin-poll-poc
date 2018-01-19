package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/plugin/rpcplugin"
	poll_model "github.com/matterpoll/matterpoll/server/model"
	"github.com/matterpoll/matterpoll/server/store"
)

var (
	// PollID is valid for alpha-numeric and hyphen(-)
	voteRoute = regexp.MustCompile(`/polls/([0-9a-z-]+)/answers/([0-9]+)/vote`)
	endPollRoute = regexp.MustCompile(`/polls/([0-9a-z-]+)/end`)
)

type MatterPollPlugin struct{
	api plugin.API
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
		return nil, model.NewAppError("here", "NilPoll", map[string]interface{}{}, "", 1)
	}
	if err = p.store.CreatePoll(poll); err != nil {
		return nil, model.NewAppError("here", "CreatePoll", map[string]interface{}{}, err.Error(), 1)
	}
	resp := poll.ToCommandResponseJson(args.SiteURL)
	log.Printf("%v", resp.ToJson())
	return resp, nil
}

func responseIntegration(w http.ResponseWriter, resp string){
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, resp)
}
 
func (p *MatterPollPlugin) handleVote(w http.ResponseWriter, r *http.Request) {
	log.Println("Handle Vote")
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	var integrationReq model.PostActionIntegrationRequest
	if err := decoder.Decode(&integrationReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		responseIntegration(w, err.Error())
		return
	}

	ret := voteRoute.FindAllStringSubmatch(r.URL.Path, 1)
	pollId := ret[0][1]
	voteId := ret[0][2]
	updated, err := p.store.Vote(pollId, voteId, integrationReq.UserId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		responseIntegration(w, err.Error())
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
	b, e := json.Marshal(resp)
	if e != nil {
		w.WriteHeader(http.StatusInternalServerError)
		responseIntegration(w, e.Error())
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
		responseIntegration(w, err.Error())
		return
	}

	matches := endPollRoute.FindAllStringSubmatch(r.URL.Path, 1)
	pollId := matches[0][1]
	log.Printf("EndPollID: %s", pollId)

	answers, err := p.store.ReadPollAnswers(pollId)
	log.Printf("Answers: %v", answers)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		responseIntegration(w, err.Error())
		return
	}
	summary := map[string][]string{}
	for user, answer := range answers {
		if v, ok := summary[answer]; ok {
			log.Printf("  - add summary: %v - %v", answer, user)
			summary[answer] = append(v, user)
		} else {
			log.Printf("  - new summary: %v - %v", answer, user)
			summary[answer] = []string{user}
		}
	}
	log.Printf("%v", summary)

	poll, err := p.store.ReadPoll(pollId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		responseIntegration(w, err.Error())
		return
	}

	var ret []string
	for _, op := range poll.Options {
		if v, ok := summary[op.ID]; ok {
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
	b, e := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		responseIntegration(w, e.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}


func (p *MatterPollPlugin) OnActivate(api plugin.API) error {
	p.api = api
	p.store = store.NewMatterPollStore(api.KeyValueStore())
	command := &model.Command{
		Trigger: "matterpoll",
		AutoComplete: true,
		AutoCompleteDesc: "sample",
		AutoCompleteHint: "sample",
	}
	if err := p.api.RegisterCommand(command); err != nil {
		return err
	}
	return nil
}

func main() {
	rpcplugin.Main(&MatterPollPlugin{})
}