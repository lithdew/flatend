import {promisify} from "util";
import dns from "dns";
import assert from "assert";

export const resolve = async (addr: string): Promise<[string, number]> => {
    const [host, port] = addr.trim().split(":");

    const resolvedHost = (await promisify(dns.lookup)(host)).address;

    const resolvedPort = parseInt(port);
    assert(resolvedPort >= 0 && resolvedPort < 65536);

    return [resolvedHost, resolvedPort];
}