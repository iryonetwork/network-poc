package eos

import (
	eos "github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/system"
)

type resources struct {
	ram, cpu, net int64
}

func (s *Storage) checkBuyResources(user string) error {
	res, err := s.api.GetAccount(eos.AN(user))
	if err != nil {
		return err
	}

	available := parseAvailableResources(res)

	toBuy := getMissingResources(available)
	if toBuy.cpu > 0 || toBuy.net > 0 {
		if err = s.buyCpuNet(user, toBuy.cpu, toBuy.net); err != nil {
			return err
		}
	}
	if toBuy.ram > 0 {
		err = s.buyRAM(user, uint32(toBuy.ram))
	}
	return err
}

func parseAvailableResources(response *eos.AccountResp) *resources {
	out := &resources{}

	// Get cpu
	out.cpu = int64(response.CPULimit.Available)
	// Get net
	out.net = int64(response.NetLimit.Available)
	// Get ram
	out.ram = response.RAMQuota - response.RAMUsage

	return out
}

// TODO calculate based on current resources price
func getMissingResources(available *resources) *resources {
	missing := &resources{}
	if available.cpu < 30000 {
		missing.cpu = (30000 - available.cpu)
	}
	if available.net < 5000 {
		missing.net = 5000 - available.net
	}
	if available.ram < 10000 {
		missing.ram = 5000 - available.ram
	}
	return missing
}

func (s *Storage) buyRAM(user string, amount uint32) error {
	s.log.Debugf("Buying %d bytes of RAM for %s", amount, user)
	_, err := s.api.SignPushActions(system.NewBuyRAMBytes(eos.AN(s.config.EosAccount), eos.AN(user), amount))
	return err
}

func (s *Storage) buyCpuNet(user string, cpuVal, netVal int64) error {
	s.log.Debugf("Buying %d CPU and %d NET for %s", cpuVal, netVal, user)
	_, err := s.api.SignPushActions(s.buyCpuNetRequest(s.config.EosAccount, cpuVal, netVal))
	return err
}

func (s *Storage) buyCpuNetRequest(account string, cpuVal, netVal int64) *eos.Action {
	transfer := false
	if account != s.config.EosAccount {
		transfer = true
	}
	return system.NewDelegateBW(eos.AN(s.config.EosAccount), eos.AN(account), eos.NewEOSAsset(cpuVal), eos.NewEOSAsset(netVal), transfer)
}
