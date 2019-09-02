# mdnstool

An extrmemely simple utility for performing discovery of local mDNS services, as well as announcing new ones to the local network.

# What is mDNS?

mDNS stands for [_Multicast DNS_](https://en.wikipedia.org/wiki/Multicast_DNS), and is a network protocol that allows applications to "discover" other applications on a local network by using IP multicast packets to request details from any participating servers directly. This allows for services to announce themselves and be discovered by other devices on the network without setting up a dedicated DNS server.

# Why this tool?

After not-a-lot of searching, I wasn't able to find a simple command-line utility that allowed me to easily announce and discover mDNS services. I was also interested in a programmatic way of adding mDNS capabilities to my other Golang projects, so wrote this utility as a debugger and a way to play with the mDNS library I found.

Additionally, `mdnstool` also contains a useful feature in that it can act as a DNS resolver for discovered mDNS hosts. For example, you could run the tool as `mdnstool discover -D :53`, which will perform continuous mDNS host discovery and liveness checking, and _also_ start a DNS server on
port 53. This server will respond to A and SRV requests and respond with the the address of the discovered service. One or more instances of this server running on a network can provide a very simple DNS-based service discovery mechanism.

# Installation

## Binaries

Check the [Releases](https://github.com/ghetzel/mdnstool/releases) for pre-compiled binaries.

## From Source

```
go get github.com/ghetzel/mdnstool
```
