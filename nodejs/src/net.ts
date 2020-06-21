import { IPv4, IPv6 } from "ipaddr.js";
import net from "net";
import events from "events";
import ipaddr from "ipaddr.js";

/**
 * Returns an available TCP host/port that may be listened to.
 */
export async function getAvailableAddress(): Promise<{
  family: string;
  host: IPv4 | IPv6;
  port: number;
}> {
  const server = net.createServer();
  server.unref();
  server.listen();

  await events.once(server, "listening");

  const info = (<net.AddressInfo>server.address())!;

  let host = ipaddr.parse(info.address.length === 0 ? "0.0.0.0" : info.address);
  if (host.kind() === "ipv6" && (<IPv6>host).isIPv4MappedAddress()) {
    host = (<IPv6>host).toIPv4Address();
  }

  server.close();

  await events.once(server, "close");

  return { family: info.family, host, port: info.port };
}

export function splitHostPort(
  addr: string
): { host: IPv4 | IPv6; port: number } {
  const fields = addr.split(":").filter((field) => field.length > 0);
  if (fields.length === 0)
    throw new Error("Unable to split host:port from address.");

  const port = parseInt(fields.pop()!);
  if (port < 0 || port > 2 ** 16) throw new Error(`Port ${port} is invalid.`);

  let host = ipaddr.parse(fields.length === 0 ? "0.0.0.0" : fields.join(":"));
  if (host.kind() === "ipv6" && (<IPv6>host).isIPv4MappedAddress()) {
    host = (<IPv6>host).toIPv4Address();
  }

  return { host, port };
}
