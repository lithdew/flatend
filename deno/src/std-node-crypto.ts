import { Buffer } from "https://deno.land/std/node/buffer.ts";

export default {createCipheriv, createDecipheriv}

// export function createCipheriv(
//   algorithm: CipherCCMTypes,
//   key: CipherKey,
//   iv: BinaryLike | null,
//   options: CipherCCMOptions
// ): CipherCCM
// export function createCipheriv(
//   algorithm: CipherGCMTypes,
//   key: CipherKey,
//   iv: BinaryLike | null,
//   options?: CipherGCMOptions
// ): CipherGCM
export function createCipheriv(
  algorithm: string,
  key: CipherKey,
  iv: BinaryLike | null,
  options?: any//stream.TransformOptions
): Cipher{
  return new Cipher()
}

class Cipher /*extends stream.Transform */{
  // private constructor()
  update(data: BinaryLike): Buffer{
          console.log('todo: crypto.Cipher.update')
    return Buffer.from([])
  }
  // update(data: string, input_encoding: Utf8AsciiBinaryEncoding): Buffer
  // update(
  //   data: NodeJS.ArrayBufferView,
  //   input_encoding: undefined,
  //   output_encoding: HexBase64BinaryEncoding
  // ): string
  // update(
  //   data: string,
  //   input_encoding: Utf8AsciiBinaryEncoding | undefined,
  //   output_encoding: HexBase64BinaryEncoding
  // ): string
  final(): Buffer {
          console.log('todo: crypto.Cipher.update')
    return Buffer.from([])
  }
  // final(output_encoding: BufferEncoding): string
  // setAutoPadding(auto_padding?: boolean): this
  getAuthTag(): Buffer {
          console.log('todo: crypto.Cipher.update')
    return Buffer.from([])
  }
  // setAAD(buffer: Buffer): this; // docs only say buffer
}


export    function createDecipheriv(algorithm: string, key: CipherKey, iv: BinaryLike | null, options?:any/* stream.TransformOptions*/): Decipher{
  return new Decipher()
}


    class Decipher /* extends stream.Transform*/ {
        // private constructor();
        update(data: BinaryLike): Buffer{
          console.log('todo: crypto.Decipher.update')
          return Buffer.from([])
        }
        // update(data: NodeJS.ArrayBufferView): Buffer;
        // update(data: string, input_encoding: HexBase64BinaryEncoding): Buffer;
        // update(data: NodeJS.ArrayBufferView, input_encoding: HexBase64BinaryEncoding | undefined, output_encoding: Utf8AsciiBinaryEncoding): string;
        // update(data: string, input_encoding: HexBase64BinaryEncoding | undefined, output_encoding: Utf8AsciiBinaryEncoding): string;
        final(): Buffer {
          console.log('todo: crypto.Decipher.final')
          return Buffer.from([])
        }
        // final(output_encoding: BufferEncoding): string;
        // setAutoPadding(auto_padding?: boolean): this;
        setAuthTag(tag: Buffer): this {
          console.log('todo: crypto.Decipher.setAuthTag')
          return this
        }
        // setAAD(buffer: NodeJS.ArrayBufferView): this;
    }

// Types

    // type KeyObjectType = 'secret' | 'public' | 'private';

    // interface KeyExportOptions<T extends KeyFormat> {
    //     type: 'pkcs1' | 'spki' | 'pkcs8' | 'sec1';
    //     format: T;
    //     cipher?: string;
    //     passphrase?: string | Buffer;
    // }

    // class KeyObject {
    //     private constructor();
    //     asymmetricKeyType?: KeyType;
    //     *
    //      * For asymmetric keys, this property represents the size of the embedded key in
    //      * bytes. This property is `undefined` for symmetric keys.
         
    //     asymmetricKeySize?: number;
    //     export(options: KeyExportOptions<'pem'>): string | Buffer;
    //     export(options?: KeyExportOptions<'der'>): Buffer;
    //     symmetricKeySize?: number;
    //     type: KeyObjectType;
    // }

    type CipherCCMTypes = 'aes-128-ccm' | 'aes-192-ccm' | 'aes-256-ccm' | 'chacha20-poly1305';
    type CipherGCMTypes = 'aes-128-gcm' | 'aes-192-gcm' | 'aes-256-gcm';

    type BinaryLike = string | Uint8Array// | NodeJS.ArrayBufferView;

    type CipherKey = BinaryLike// | KeyObject;

    interface CipherCCMOptions /*extends stream.TransformOptions*/ {
        authTagLength: number;
    }
    interface CipherGCMOptions /*extends stream.TransformOptions*/ {
        authTagLength?: number;
    }