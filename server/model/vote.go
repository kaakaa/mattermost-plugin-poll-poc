package model


type Vote map[string]string

func NewVoteSet() Vote {
	return map[string]string{}
}