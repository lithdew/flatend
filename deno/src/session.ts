import net from "net";
import nacl from "tweetnacl";
import events from "events";
import crypto from "crypto";

const blake2b = require("blake2b");

export function x25519(privateKey: Uint8Array, publicKey: Uint8Array): Buffer {
  return blake2b(32).update(nacl.scalarMult(privateKey, publicKey)).digest();
}

export async function serverHandshake(conn: net.Socket): Promise<Uint8Array> {
  const serverKeys = nacl.box.keyPair();

  await events.once(conn, "readable");
  const clientPublicKey = conn.read(nacl.box.publicKeyLength);

  conn.write(serverKeys.publicKey);

  return x25519(serverKeys.secretKey, clientPublicKey);
}

export async function clientHandshake(client: net.Socket): Promise<Uint8Array> {
  const clientKeys = nacl.box.keyPair();

  client.write(clientKeys.publicKey);

  await events.once(client, "readable");
  const serverPublicKey = client.read(nacl.box.publicKeyLength);

  return x25519(clientKeys.secretKey, serverPublicKey);
}

export class Session {
  secret: Uint8Array;
  readNonce: bigint = BigInt(0);
  writeNonce: bigint = BigInt(0);

  constructor(secret: Uint8Array) {
    this.secret = secret;
  }

  public async *decrypted(stream: AsyncIterable<Buffer>) {
    for await (const frame of stream) {
      yield this.decrypt(frame);
    }
  }

  public encrypt(src: string | ArrayBufferLike): Buffer {
    const buf = Buffer.isBuffer(src) ? src : Buffer.from(src);

    const nonce = Buffer.alloc(12);
    nonce.writeBigUInt64BE(this.writeNonce);
    this.writeNonce = this.writeNonce + BigInt(1);

    const cipher = crypto.createCipheriv("aes-256-gcm", this.secret, nonce, {
      authTagLength: 16,
    });
    const ciphered = cipher.update(buf);
    cipher.final();

    return Buffer.concat([ciphered, cipher.getAuthTag()]);
  }

  public decrypt(buf: Buffer): Buffer {
    if (buf.byteLength < 16) throw new Error("Missing authentication tag.");

    const nonce = Buffer.alloc(12);
    nonce.writeBigUInt64BE(this.readNonce);
    this.readNonce = this.readNonce + BigInt(1);

    const decipher = crypto.createDecipheriv(
      "aes-256-gcm",
      this.secret,
      nonce,
      { authTagLength: 16 }
    );
    decipher.setAuthTag(buf.slice(buf.byteLength - 16, buf.byteLength));

    const deciphered = decipher.update(buf.slice(0, buf.byteLength - 16));
    decipher.final();

    return deciphered;
  }
}
