package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"

	pollmodel "github.com/matterpoll/matterpoll/server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/model"
	"github.com/satori/go.uuid"
)

const (
	PollKeyFormat = "poll_%s"
	PollOptionKeyFormat = "poll_%s_vote"
)

type MatterPollStore struct {
	store plugin.KeyValueStore
}

func NewMatterPollStore(store plugin.KeyValueStore) *MatterPollStore {
	return &MatterPollStore {
		store: store,
	}
}

func (s *MatterPollStore)CreatePoll(poll *pollmodel.Poll) *model.AppError {
	// Create MatterPoll store
	id := uuid.NewV4()
	poll.ID = id.String()
	b, e := json.Marshal(poll)
	if e != nil {
		return model.NewAppError("CreatePoll", "json.Marshal error1", nil, e.Error(), 0)
	}
	key := fmt.Sprintf(PollKeyFormat, poll.ID)
	if err := s.store.Set(key, b); err != nil {
		return model.NewAppError("CreatePoll", "set key error", nil, err.Error(), 0)
	}

	// Store initial vote result
	vote := pollmodel.NewVoteSet()
	initVote, e := json.Marshal(&vote)
	if e != nil {
		return model.NewAppError("CreatePoll", "json.Marshal error2", nil, e.Error(), 0)
	}
	key = fmt.Sprintf(PollOptionKeyFormat, poll.ID)
	if err := s.store.Set(key, initVote); err != nil {
		return model.NewAppError("CreatePoll", "set init vote error", nil, err.Error(), 0)
	}
	return nil
}

func (s *MatterPollStore) ReadPoll(pollId string) (*pollmodel.Poll, *model.AppError) {
	key := fmt.Sprintf(PollKeyFormat, pollId)
	b, err := s.store.Get(key)
	if err != nil {
		return nil, model.NewAppError("ReadPoll", "", nil, err.Error(), 0)
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	var poll pollmodel.Poll
	if err := decoder.Decode(&poll); err != nil {
		return nil, model.NewAppError("ReadPoll", "", nil, err.Error(), 0)
	}
	return &poll, nil
}

func (s *MatterPollStore) ReadPollAnswers(pollId string) (pollmodel.Vote, *model.AppError) {
	key := fmt.Sprintf(PollOptionKeyFormat, pollId)
	b, err := s.store.Get(key)
	if err != nil {
		return nil, model.NewAppError("ReadPollAnswer", "", nil, err.Error(), 0)
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	var vote pollmodel.Vote
	if err := decoder.Decode(&vote); err != nil {
		return nil, model.NewAppError("ReadPollAnswer", "", nil, err.Error(), 0)
	}
	return vote, nil
}

func (s *MatterPollStore)Vote(pollId, optionId, userId string) (bool, *model.AppError) {
	// Read vote count
	vote, err := s.GetVotes(pollId, optionId)
	log.Printf("Existing vote: %v", vote)

	if err != nil {
		return false, model.NewAppError("Vote", "", nil, err.Error(), 0)
	}

	// Increment vote count
	_, updated := vote[userId]
	vote[userId] = optionId

	log.Printf("Updated vote: %v", vote)
	b, e := json.Marshal(vote)
	if e != nil {
		return false, model.NewAppError("Vote", "", nil, e.Error(), 0)
	}
	key := fmt.Sprintf(PollOptionKeyFormat, pollId)
	if err = s.store.Set(key, b); err != nil {
		return false, model.NewAppError("Vote", "", nil, err.Error(), 0)
	}
	return updated, nil
}

func (s *MatterPollStore) GetVotes(pollId, optionId string) (pollmodel.Vote, *model.AppError) {
	key := fmt.Sprintf(PollOptionKeyFormat, pollId)
	b, err := s.store.Get(key)
	if err != nil {
		return nil, model.NewAppError("GetVotes", "", nil, err.Error(), 0)
	}

	decoder := json.NewDecoder(bytes.NewReader(b))
	var vote pollmodel.Vote
	if err := decoder.Decode(&vote); err != nil {
		return nil, model.NewAppError("GetVotes", "", nil, err.Error(), 0)
	}
	return vote, nil
}

func (s *MatterPollStore) DeletePoll(poll pollmodel.Poll) *model.AppError {
	key := fmt.Sprintf(PollKeyFormat, poll.ID)
	if err := s.store.Delete(key); err != nil {
		return model.NewAppError("DeletePoll", "", nil, err.Error(), 0)
	}

	for _, op := range poll.Options {
		key = fmt.Sprintf(PollOptionKeyFormat, poll.ID, op.ID)
		if err := s.store.Delete(key); err != nil {
			log.Printf("Delete Poll Error: %s", key)
			return model.NewAppError("DeletePoll", "", nil, err.Error(), 0)
		}
	}
	return nil
}