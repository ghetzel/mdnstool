package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/netutil"
	"github.com/miekg/dns"
)

type service struct {
	netutil.Service
	LastSeen time.Time
}

func (self *service) FQDN() string {
	return dk(self.Instance + `.` + self.Service.Service + self.Domain)
}

type serviceList []*service

var DefaultTTL = 10 * time.Second
var DefaultServiceLifetime = 10 * time.Second

func (self serviceList) ContainsService(svc *service) bool {
	if svc == nil {
		return true
	}

	for _, existing := range self {
		if existing.String() == svc.String() {
			existing.LastSeen = time.Now()
			log.Debugf("[dns] service %q was seen", existing.String())

			return true
		}
	}

	return false
}

func (self serviceList) FilterByHostname(fqdn string) serviceList {
	out := make(serviceList, 0)

	for _, svc := range self {
		if svc.FQDN() == dk(fqdn) {
			out = append(out, svc)
		}
	}

	return out
}

type DNS struct {
	TTL             time.Duration
	ServiceLifetime time.Duration
	server          *dns.Server
	services        serviceList
	zco             *netutil.ZeroconfOptions
	running         bool
	svclock         sync.Mutex
}

func NewDNS(address string, zeroconfOptions *netutil.ZeroconfOptions) *DNS {
	return &DNS{
		TTL:             DefaultTTL,
		ServiceLifetime: DefaultServiceLifetime,
		services:        make(serviceList, 0),
		zco:             zeroconfOptions,
		running:         true,
		server: &dns.Server{
			Addr: address,
			Net:  `udp`,
		},
	}
}

// Start continuous mDNS discovery and then start a DNS server to serve what we find.
func (self *DNS) ListenAndServe() error {
	if self.server.Handler == nil {
		self.server.Handler = self
	}

	if self.zco != nil {
		self.zco.Limit = 0
		self.zco.Timeout = self.ServiceLifetime

		go func() {
			for self.running {
				if err := netutil.ZeroconfDiscover(self.zco, func(svc *netutil.Service) bool {
					self.svclock.Lock()
					defer self.svclock.Unlock()

					s := &service{
						Service:  *svc,
						LastSeen: time.Now(),
					}

					if !self.services.ContainsService(s) {
						self.services = append(self.services, s)
						log.Debugf("[dns] discovery: appended %v", svc)
					}

					return true
				}); err != nil {
					log.Fatalf("discovery error: %v", err)
				}
			}
		}()

		log.Infof("[dns] Starting DNS server at %s", self.server.Addr)
		return self.server.ListenAndServe()
	} else {
		return fmt.Errorf("No mDNS configuration provided")
	}
}

// Handle an individual DNS query.
func (self *DNS) ServeDNS(w dns.ResponseWriter, req *dns.Msg) {
	self.svclock.Lock()
	defer self.svclock.Unlock()

	msg := new(dns.Msg)
	msg.SetReply(req)

	var domain string

	log.Debugf("[dns] QUERY:")

	for _, q := range req.Question {
		log.Debugf("[dns]   %s", strings.TrimPrefix(q.String(), `;`))
		domain = q.Name

		switch qtype := q.Qtype; qtype {
		case dns.TypeA, dns.TypeAAAA, dns.TypeSRV:
			break
		default:
			dns.HandleFailed(w, req)
			log.Debugf("[dns] ANSWER: SERVFAIL")
			return
		}

		if domain != `` {
			for _, svc := range self.services.FilterByHostname(domain) {
				if time.Since(svc.LastSeen) > self.ServiceLifetime {
					go self.removeService(svc)
					continue
				}

				msg.Authoritative = true

				for _, addr := range svc.Addresses {
					var answer dns.RR

					hdr := dns.RR_Header{
						Name:   dk(domain),
						Rrtype: q.Qtype,
						Class:  dns.ClassINET,
						Ttl:    uint32(self.TTL.Round(time.Second).Seconds()),
					}

					// reply with the first routable IP the service published
					if ipv4 := netutil.IsRoutableIP(`ip4`, addr); ipv4 != nil {
						switch q.Qtype {
						case dns.TypeA:
							answer = &dns.A{
								A:   ipv4,
								Hdr: hdr,
							}
						case dns.TypeSRV:
							answer = &dns.SRV{
								Priority: 0,
								Weight:   5,
								Port:     uint16(svc.Port),
								Target:   svc.Hostname,
								Hdr:      hdr,
							}
						}
					} else if ipv6 := netutil.IsRoutableIP(`ip6`, addr); ipv6 != nil {
						switch q.Qtype {
						case dns.TypeAAAA:
							answer = &dns.AAAA{
								AAAA: ipv6,
								Hdr:  hdr,
							}
						}
					}

					if answer != nil {
						msg.Answer = append(msg.Answer, answer)
						break
					}
				}
			}
		}
	}

	if len(msg.Answer) > 0 {
		log.Debugf("[dns] ANSWER:")

		for _, a := range msg.Answer {
			log.Debugf("[dns]   %v", a)
		}

		w.WriteMsg(msg)
	} else {
		dns.HandleFailed(w, req)
		log.Debugf("[dns] ANSWER: SERVFAIL")
	}
}

func (self *DNS) removeService(svc *service) {
	self.svclock.Lock()
	defer self.svclock.Unlock()

	for i, s := range self.services {
		if s == svc {
			self.services = append(self.services[:i], self.services[i+1:]...)
			log.Infof("[dns] service %q was removed", svc.String())
		}
	}
}

func dk(in string) string {
	in = strings.TrimSuffix(in, `.`)
	return in + `.`
}
