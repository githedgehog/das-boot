# Alpha2 Chain Booting TODOs

WIP items for chainbooting.
NOTE: these are not necessarily complete or still accurate!

- Change to hhfab for das-boot development
- chain booting: Store CA certificates for controller root of trust <https://github.com/githedgehog/das-boot/issues/113>
- chain booting: rework DynLL listener - rework dynll listener to query attached switches based on agent objects only. Must also start/stop listeners based on changes to agents, so we must introduce watches for agent objects
- chain booting: rework staged installers to query everything over IPv6 Link Local address of the neighbour that was answering the original request
  - redo configurations of installers (a lot of options can go away)
  - rework DNS part: either write all required entries out to /etc/hosts, or run a DNS server within the installer which gets written out to /etc/resolv.conf
  - rework syslog: will get sent to IPv6 Link Local of the neighbour (either change in configuration, or hard code that behaviour)
  - rework NTP: will get sent to IPv6 link local of the neighbour (either change in configuration, or hard code that behaviour)
  - remove IPAM altogether: not required anymore when everything is happening over the IPv6 link-local address

- chain booting: implement TCP proxy for ports 80 and 443
  - needs to listen on IPv6 link local addresses of the right interfaces
  - needs to forward connections to the right ports on the seeder / control VIP based on agent configuration
- chain booting: implement UDP proxy for syslog
  - needs to listen on IPv6 link local addresses of the right interfaces
  - needs to forward syslog packets to the node port of the syslog service on the control VIP
- chain booting: implement UDP proxy for NTP server
  - needs to listen on IPv6 link local addresses of the right interfaces
  - needs to forward NTP requests to the node port of the NTP service on the control VIP
  - must be able to receive UDP responses for the NTP requests from the NTP service
  - and must forward them to the original source on the right IPv6 link local interface

Boot from Front Panel Ports TODOs:

- platform specific library to interact with SFPs <https://github.com/githedgehog/saictl/issues/6>
- "ports" subcommand which gives a general overview of the status of all ports <https://github.com/githedgehog/saictl/issues/7>
- access to SAI vendor shell through subcommand <https://github.com/githedgehog/saictl/issues/10>
- port detection routine which tries to bring up ports which must be switchable through saictl <https://github.com/githedgehog/saictl/issues/9>
- netlink monitor loop which adds/removes routes in asic <https://github.com/githedgehog/saictl/issues/12>
- add additional required traps for IPv6 link local networking to make neighbour discovery work
- add additional required traps for DHCP for IPv4
- 52xx: rework onie-syseeprom <https://github.com/githedgehog/honie/issues/26>
- 52xx: package ONIE SAI tooling <https://github.com/githedgehog/honie/issues/29>
- enabling of reverting to original ONIE <https://github.com/githedgehog/honie/issues/17>
- packaging HONIE as NOS installer for ZTP installation on factory-default switches from USB

- add "honie" specific commands for enabling/disabling HONIE features <https://github.com/githedgehog/honie/issues/16>
- Add HONIE release information <https://github.com/githedgehog/honie/issues/15>
- "sai" subcommand within saictl which interacts with SAI <https://github.com/githedgehog/saictl/issues/5>
- "xcvr" subcommand which interacts with the platform specific parts for the transceivers <https://github.com/githedgehog/saictl/issues/8>
- 52xx: make disabling of front panel ports optional <https://github.com/githedgehog/honie/issues/30>

## Rethinking Chainbooting after IPv6 link-local discovery fails in SONiC

ok ... so I reviewed everything again today ... here is how it is

we are currently relying on IPv6 link-local in das boot for two reasons:

- when we make the IPAM request, we can tell by the original installer URL (because it was downloaded over IPv6 link-local) over which link it was downloaded, and we pass the interface to the IPAM request, and on the server-side we can then tell which IP configurations it needs to serve
- for security reasons from a mid-term / long-term point of view: it locks down the installer which must be considered an untrusted environment at that point that it can never hop further past the next link, and it can perform communication only over link-local until it becomes an approved device

reevaluating this all again, here is what I'm thinking:

- the IPAM request is going away anyways with the current idea that everything is always done over the next hop IPv6 link-local IP
- in general in retrospect the IPAM request was probably strictly speaking not entirely necessary, things could have been sent just in the config with stage0
- that said, we were sending along the device ID, and I remember that I originally wanted to do something with that at that stage already, but it's not implemented anyways

so we essentially have the following options I think:

(A) we implement neighbor discovery through LLDP

- for this to work we need to ensure that on all SONiC instances LLDP is enabled
- for all relevant ports where we know that switches are attached we enable IPv6 on the port which gives us an IPv6 link-local address for the port
- we configure LLDP for that port in SONiC to include the IPv6 link-local address of that link to be included as a management IP address in LLDP
- in the onie-said we add an LLDP listener which collects and reads LLDP packets and extracts all IPv6 link-local addresses from the management IPs of the LLDP packets
- in the onie-saictl we add a command which can extract these IPs per interface
- we enhance ONIE to do an "LLDP Neighbour Discovery" which we will perform before its standard "all nodes" mulitcast ping discovery, which essentially just reads the IP addresses for that interface by using onie-saictl which will actually be faster than what ONIE is doing for the multicast ping as Sergei can attest to can take a long time :)
- besides from that we continue with the plan as we have it with IPv6 link-local proxies at the next switch

(B) considering using DHCP with proxy at the next switch, here is what we would need to do:

- DHCP server needs to identify the request as coming from a specific link on the switch, and issue a dedicated /31 per link per switch.
- Note that per switch is not enough as there can be multiple links between the two devices and with a standard global-scoped IPv4 addresses that would produce a conflict otherwise, but I think we have that covered
- the DHCP server must serve the "default-url" option as being the other side of the /31 where we run the proxy
- the rest would probably already continue as we have it currently planned
  - we get rid of the IPAM request
  - we either serve the additional information from the IPAM request like syslog and NTP servers in the stage0 configuration, or we simply hard-code them now, as they will always point to the next hop
  - the proxy forwards the NTP and syslog, and maps http/https to the right port on the seeder for the switch

(C) thinking about what Sergei said that we could potentially eliminate the proxies altogether, here is what we would need to do:

- for DHCP address allocation the same applies as above, however, there are different options now
- the DHCP server must send the "default-url" as being the mapped port location for that particular switch
- the DHCP server must send the control VIP as a static route through option 33. however, the chances that ONIE is going to respect that option and will actually apply it are slim
- which means that we would need send a default gateway option, which would probably break for mulitple links because I doubt that there is sophisticated handling for that, so this is no long-term solution
- from here on, we could continue like the following:
  - we could still get rid of the IPAM request, or we could simply leave it as-is (saves time)
  - if we get rid of it, we must move now all the information from the IPAM request into stage0 config
- as there are no proxies now, we essentially continue as we do right now:
  - we configure DNS as right now
  - we do syslog directly as right now
  - we do NTP directly as right now
  - everything continues the same
- one benefit here is that the traffic is actually not going to hit the CPU of the next switch

here are my opinions about these plans:

- (A) is the direct workaround for the problem at hand. Because we cannot change SONiC to accomodate the multicast ping because we're using Broadcom SONiC this is a feasible solution. Considering that we shouldn't let multicast traffic going through to the CPU anyways, this is not a bad solution. It also means we're not deviating from the plan, and we keep it "DHCP-free" which would still be a good thing IMHO
- (B) is probably the long-term solution that we are going to end up with: we need the proxies for other things like image-caching etc. in the future anyways. Using DHCP would speed up things within ONIE as well, as this is the preferred method anyways, and it just follows the KISS principle. I like it for that. What I don't like here is the global IP addressing and routing though. However, there are solutions for this. Nonetheless this part just does not sit well with me, and I hope it will not destroy our security story.
- (C) is problematic as mentioned above for multiple reasons (multiple links, global IP addressing, direct access to the seeder on the right port which could easily be changed, removes the caching possibility in the future etc., overall security in general). However, for the upcoming release it could serve us well, and it could give us some time back. Also, that it bypasses the CPU on the next switch isn't necessarily a bad thing either.

So (A) is probably my favourite solution. However, I'm concerned that Sergei has mentioned that he had issues with LLDP before. And as we cannot fix anything in SONiC, this might currently not be a good idea.

Considering what I just said about (A), (B) is probably the right path in the longer run as a solid alternative.

And (C) is probably the best option for our upcoming release. It saves us time, we can think over (A) and (B) again, and push the real implementation out by a release, and use the time to speed up things for this release hopefully which are not optional.


More TODOs:

- comment in https://docs.google.com/document/d/1Gz7iNtJNMI-zKJhaOcI3aflPCx3etJ01JMxzbtvruKk/edit
- we should merge: https://github.com/sonic-net/sonic-buildimage/pull/17024 into our saibcm-modules
- we can potentially move and upgrade the Linux kernel to 6.1 (check out main sonic-linux-kernel repo which switched already)
- redo our saibcm-modules package so that we have that in git repository
