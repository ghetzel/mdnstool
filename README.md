# mdnstool

An extrmemely simple utility for performing discovery of local mDNS services, as well as announcing new ones to the local network.

# What is mDNS?

mDNS stands for [_Multicast DNS_](https://en.wikipedia.org/wiki/Multicast_DNS), and is a network protocol that allows applications to "discover" other applications on a local network by using IP multicast packets to request details from any participating servers directly. This allows for services to announce themselves and be discovered by other devices on the network without setting up a dedicated DNS server.

# Why this tool?

After not-a-lot of searching, I wasn't able to find a simple command-line utility that allowed me to easily announce and discover mDNS services. I was also interested in a programmatic way of adding mDNS capabilities to my other Golang projects, so wrote this utility as a debugger and a way to play with the mDNS library I found.

# Installation
