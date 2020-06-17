import { promisify } from "util";
import dns from "dns";
import assert from "assert";
import ipaddr, { IPv4, IPv6 } from "ipaddr.js";

export const resolve = async (addr: string): Promise<[IPv4 | IPv6, number]> => {
  const fields = addr.trim().split(":");

  const host = fields.slice(0, fields.length - 1).join(":");
  const port = fields.pop()!;

  let resolvedHost = ipaddr.parse((await promisify(dns.lookup)(host)).address);
  if (
    resolvedHost.kind() === "ipv6" &&
    (<IPv6>resolvedHost).isIPv4MappedAddress()
  ) {
    resolvedHost = (<IPv6>resolvedHost).toIPv4Address();
  }

  const resolvedPort = parseInt(port);
  assert(resolvedPort >= 0 && resolvedPort < 65536);

  return [resolvedHost, resolvedPort];
};
