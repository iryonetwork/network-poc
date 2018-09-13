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
	t := &TokenList{make(map[string]*token), log}

	go func() {
		for {
			time.Sleep(viableFor)
			for id, token := range t.tokens {
				if token.viableUntil.Before(time.Now()) {
					log.Debugf("Removing token: %s", id)
					err := t.RevokeToken(id)
					if err != nil {
						log.Fatalf("Error removing tokens: %v", err)
					}
				}
			}
		}
	}()

	return t
}

func (t *TokenList) NewToken(id string, exists bool) (string, time.Time, error) {
	// Generate new token
	tok, err := uuid.NewV4()
	if err != nil {
		return "", time.Now(), err
	}
	viableUntil := time.Now().Add(viableFor)

	// Add token to storage
	t.tokens[tok.String()] = &token{exists, id, viableUntil}

	return tok.String(), viableUntil, nil
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

func (t *TokenList) ValidateGetInfo(tok string) (string, bool) {
	if t.exists(tok) {
		return t.getID(tok), true
	}
	return "", false
}

func (t *TokenList) RevokeToken(tok string) error {
	if _, ok := t.tokens[tok]; !ok {
		return fmt.Errorf("Token does not exists")
	}
	delete(t.tokens, tok)
	return nil
}

func (t *TokenList) exists(tok string) bool {
	if _, ok := t.tokens[tok]; ok {
		return true
	}
	return false
}

func (t *TokenList) getID(tok string) string {
	return t.tokens[tok].id
}
