package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/go-stockutil/executil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/netutil"
)

func main() {
	app := cli.NewApp()
	app.Name = `mdnstool`
	app.Usage = `utility for working with mDNS host discovery`
	app.Version = `0.0.3`
	hostname, _ := os.Hostname()

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `info`,
			EnvVar: `LOGLEVEL`,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name: `discover`,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  `format, f`,
					Usage: `The output format for discovered services (one of: json, plain)`,
					Value: `plain`,
				},
				cli.StringFlag{
					Name:  `service, s`,
					Usage: `The mDNS service to browse`,
					Value: `_http._tcp`,
				},
				cli.StringFlag{
					Name:  `domain, d`,
					Usage: `The mDNS TLD to browse`,
					Value: `.local`,
				},
				cli.IntFlag{
					Name:  `limit, l`,
					Usage: `Stop looking after this number of services are found`,
				},
				cli.DurationFlag{
					Name:  `timeout, t`,
					Usage: `Stop looking after this amount of time`,
					Value: 30 * time.Second,
				},
				cli.StringFlag{
					Name:  `match-instance, I`,
					Usage: `A regular expression that mDNS service instances must match`,
				},
				cli.StringFlag{
					Name:  `match-hostname, H`,
					Usage: `A regular expression that mDNS hostnames must match`,
				},
				cli.StringFlag{
					Name:  `match-port, P`,
					Usage: `A regular expression that mDNS service ports must match`,
				},
				cli.StringFlag{
					Name:  `match-address, A`,
					Usage: `A regular expression that mDNS service addresses must match`,
				},
				cli.StringFlag{
					Name:  `dns-server, D`,
					Usage: `If this flag is provided, mdnstool will start a DNS resolver that continuously discovers mDNS services and will serve DNS responses for those hosts.`,
					Value: `:53`,
				},
			},
			Action: func(c *cli.Context) {
				opts := &netutil.ZeroconfOptions{
					Limit:         c.Int(`limit`),
					Timeout:       c.Duration(`timeout`),
					Service:       c.String(`service`),
					Domain:        c.String(`domain`),
					MatchInstance: c.String(`match-instance`),
					MatchPort:     c.String(`match-port`),
					MatchHostname: c.String(`match-hostname`),
					MatchAddress:  c.String(`match-address`),
				}

				if c.IsSet(`dns-server`) {
					log.FatalIf(
						NewDNS(c.String(`dns-server`), opts).ListenAndServe(),
					)
				} else {
					if err := netutil.ZeroconfDiscover(opts, func(svc *netutil.Service) bool {
						switch c.String(`format`) {
						case `json`:
							json.NewEncoder(os.Stdout).Encode(svc)
						default:
							fmt.Println(svc.String())
						}
						return true
					}); err != nil {
						log.Fatalf("discovery error: %v", err)
					}
				}
			},
		}, {
			Name: `publish`,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  `instance, i`,
					Usage: `The mDNS instance to announce`,
					Value: hostname,
				},
				cli.StringFlag{
					Name:  `service, s`,
					Usage: `The mDNS service to announce`,
					Value: `_http._tcp`,
				},
				cli.StringFlag{
					Name:  `domain, d`,
					Usage: `The mDNS TLD to announce`,
					Value: `.local`,
				},
				cli.IntFlag{
					Name:  `port, p`,
					Usage: `The port to announce`,
				},
				cli.StringSliceFlag{
					Name:  `txt, t`,
					Usage: `A text record entry to include with the registration`,
				},
			},
			Action: func(c *cli.Context) {
				svc := &netutil.Service{
					Instance: c.String(`instance`),
					Service:  c.String(`service`),
					Domain:   c.String(`domain`),
					Port:     c.Int(`port`),
					Text:     c.StringSlice(`txt`),
				}

				if _, err := netutil.ZeroconfRegister(svc); err == nil {
					log.Infof("[mdns] Registered service %s", svc)

					executil.TrapSignals(func(sig os.Signal) bool {
						netutil.ZeroconfUnregisterAll()
						log.Infof("[mdns] All services unregistered")
						os.Exit(0)
						return true
					}, os.Interrupt)

					select {}
				} else {
					log.Fatalf("register error: %v", err)
				}
			},
		},
	}

	app.Run(os.Args)
}
