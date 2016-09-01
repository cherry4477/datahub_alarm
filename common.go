package main

import (
	"fmt"
	log "github.com/asiainfoLDP/datahub/utils/clog"
	"github.com/miekg/dns"
	"os"
)

type dnsEntry struct {
	ip   string
	port string
}

func getEnv(name string, required bool) string {
	s := os.Getenv(name)
	if required && s == "" {
		panic("env variable required, " + name)
	}
	log.Infof("[env][%s] %s\n", name, s)
	return s
}

func dnsExchange(srvName, agentIp, agentPort string) []dnsEntry {
	Name := fmt.Sprintf("%s.service.consul", srvName)
	agentAddr := fmt.Sprintf("%s:%s", agentIp, agentPort)

	c := new(dns.Client)
	c.Net = "tcp"

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(Name), dns.TypeSRV)
	m.RecursionDesired = true

	result := []dnsEntry{}

	log.Debug("Consul addr:", agentAddr)
	r, _, err := c.Exchange(m, agentAddr)
	if r == nil {
		log.Fatalf("dns query error: %s\n", err.Error())
		return result
	}

	if r.Rcode != dns.RcodeSuccess {
		log.Fatalf("dns query error: %v\n", r.Rcode)
		return result
	}

	for _, ex := range r.Extra {
		if tmp, ok := ex.(*dns.A); ok {
			result = append(result, dnsEntry{ip: tmp.A.String()})
		}
	}

	for i, an := range r.Answer {
		if tmp, ok := an.(*dns.SRV); ok {
			port := fmt.Sprintf("%d", tmp.Port)
			result[i].port = port
		}
	}

	return result
}
