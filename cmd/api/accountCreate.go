package main

import (
	"fmt"

	"github.com/lucasjones/reggen"
)

func (s *storage) getAccName() (string, int, error) {
	s.log.Debugf("Account format set to %s", s.config.EosAccountFormat)
	g, err := reggen.NewGenerator(s.config.EosAccountFormat)
	if err != nil {
		s.log.Printf("Error generating eosname. Err: %+v", err)
		return "", 500, fmt.Errorf("Internal server error")
	}
	var accname string
	for {
		accname = g.Generate(12)
		if !s.eos.CheckAccountExists(accname) {
			break
		}
	}
	return accname, 200, nil
}

func (s *storage) newAccount(key string) (string, int, error) {
	accname, code, err := s.getAccName()
	if err != nil {
		return "", code, err
	}
	if err := s.eos.CreateAccount(accname, key); err != nil {
		fmt.Printf("Error creating new account; %+v", err)
		return "", 500, fmt.Errorf("Internal server error")
	}
	return accname, 200, nil
}
