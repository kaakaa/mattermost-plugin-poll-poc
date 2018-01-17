package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"

	pollmodel "github.com/matterpoll/matterpoll/server/model"
	"github.com/mattermost/mattermost-server/plugin"
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

func (s *MatterPollStore)CreatePoll(poll *pollmodel.Poll) error {
	// Create MatterPoll store
	id := uuid.NewV4()
	poll.ID = id.String()
	b, err := json.Marshal(poll)
	if err != nil {
		return err
	}
	key := fmt.Sprintf(PollKeyFormat, poll.ID)
	if appErr := s.store.Set(key, b); appErr != nil {
		return err
	}

	// Store initial vote result
	vote := pollmodel.NewVoteSet()
	initVote, err := json.Marshal(&vote)
	if err != nil {
		return err
	}
	key = fmt.Sprintf(PollOptionKeyFormat, poll.ID)
	if appErr := s.store.Set(key, initVote); appErr != nil {
		return err
	}
	return nil
}

func (s *MatterPollStore) ReadPoll(pollId string) (*pollmodel.Poll, error) {
	key := fmt.Sprintf(PollKeyFormat, pollId)
	b, err := s.store.Get(key)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	var poll pollmodel.Poll
	if appErr := decoder.Decode(&poll); appErr != nil {
		return nil, appErr
	}
	return &poll, nil
}

func (s *MatterPollStore) ReadPollAnswers(pollId string) (pollmodel.Vote, error) {
	key := fmt.Sprintf(PollOptionKeyFormat, pollId)
	b, err := s.store.Get(key)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	var vote pollmodel.Vote
	if appErr := decoder.Decode(&vote); appErr != nil {
		return nil, appErr
	}
	return vote, nil
}

func (s *MatterPollStore)Vote(pollId, optionId, userId string) (bool, error) {
	// Read vote count
	vote, err := s.GetVotes(pollId, optionId)
	if err != nil {
		return false, err
	}

	// Increment vote count
	_, updated := vote[userId]
	vote[userId] = optionId

	b, err := json.Marshal(vote)
	if err != nil {
		return false, err
	}
	key := fmt.Sprintf(PollOptionKeyFormat, pollId)
	if err = s.store.Set(key, b); err != nil {
		return false, err
	}
	return updated, nil
}

func (s *MatterPollStore) GetVotes(pollId, optionId string) (pollmodel.Vote, error) {
	key := fmt.Sprintf(PollOptionKeyFormat, pollId)
	b, err := s.store.Get(key)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(b))
	var vote pollmodel.Vote
	if appErr := decoder.Decode(&vote); appErr != nil {
		return nil, appErr
	}
	return vote, nil
}

func (s *MatterPollStore) DeletePoll(poll pollmodel.Poll) error {
	key := fmt.Sprintf(PollKeyFormat, poll.ID)
	if err := s.store.Delete(key); err != nil {
		return err
	}

	for _, op := range poll.Options {
		key = fmt.Sprintf(PollOptionKeyFormat, poll.ID, op.ID)
		if err := s.store.Delete(key); err != nil {
			log.Printf("Delete Poll Error: %s", key)
		}
	}
	return nil
}