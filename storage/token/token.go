package token

import (
	"fmt"
	"time"

	"github.com/iryonetwork/network-poc/logger"

	"github.com/gofrs/uuid"
)

const viableFor time.Duration = 1 * time.Hour

type token struct {
	hasAcc      bool
	id          string
	viableUntil time.Time
}
type TokenList struct {
	tokens map[string]*token
	log    *logger.Log
}

func Init(log *logger.Log) *TokenList {
	return &TokenList{make(map[string]*token), log}
}

func (t *TokenList) NewToken(id string, exists bool) (string, time.Time, error) {
	// Generate new token
	tok, err := uuid.NewV4()
	if err != nil {
		return "", time.Now(), err
	}
	viableUntil := time.Now().Add(viableFor)
	t.tokens[tok.String()] = &token{exists, id, viableUntil}
	// Revoke token after `viableFor`
	go func() {
		time.Sleep(viableFor)
		t.RevokeToken(tok.String())
		t.log.Debugf("Revoked token")
	}()
	return tok.String(), viableUntil, nil
}

func (t *TokenList) IsValid(tok string) bool {
	if _, ok := t.tokens[tok]; ok {
		return true
	}
	return false
}

func (t *TokenList) IsAccount(tok string) bool {
	return t.tokens[tok].hasAcc
}

func (t *TokenList) AccCreated(tok, account, key string) error {
	if t.tokens[tok].id != key {
		return fmt.Errorf("Token:AccCreated: Key and token key does not match")
	}
	t.tokens[tok].id = account
	t.tokens[tok].hasAcc = true
	return nil
}
func (t *TokenList) GetID(tok string) string {
	return t.tokens[tok].id
}

func (t *TokenList) RevokeToken(tok string) error {
	if _, ok := t.tokens[tok]; !ok {
		return fmt.Errorf("Token does not exists")
	}
	delete(t.tokens, tok)
	return nil
}
