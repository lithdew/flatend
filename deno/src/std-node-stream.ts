import * as events from "https://deno.land/std/node/events.ts"

export class Stream extends events.EventEmitter {
  constructor(opts?: ReadableOptions) {
    super()
    console.log('todo: stream.Stream')
  }
  pipe<T>(destination: T, options?: { end?: boolean }): T {
    return destination
  }
}


interface ReadableOptions {
  highWaterMark?: number;
  encoding?: BufferEncoding;
  objectMode?: boolean;
  read?(this: Readable, size: number): void;
  destroy?(this: Readable, error: Error | null, callback: (error: Error | null) => void): void;
  autoDestroy?: boolean;
}

export class Readable extends Stream {
  /**
   * A utility method for creating Readable Streams out of iterators.
   */
  static from(
    iterable: Iterable<any> | AsyncIterable<any>,
    options?: ReadableOptions
  ): Readable{ 
    console.log('todo: stream.Readable')
    return new Readable()}

  // readable: boolean
  // readonly readableEncoding: BufferEncoding | null
  // readonly readableEnded: boolean
  // readonly readableHighWaterMark: number
  // readonly readableLength: number
  // readonly readableObjectMode: boolean
  destroyed: boolean=false
  // constructor(opts?: ReadableOptions)
  _read(size: number): void{ console.log('TODO _read') }
  read(size?: number): any{
    console.log('todo: stream.Readable.read', size)

  }
  // setEncoding(encoding: BufferEncoding): this
  // pause(): this
  // resume(): this
  // isPaused(): boolean
  // unpipe(destination?: NodeJS.WritableStream): this
  // unshift(chunk: any, encoding?: BufferEncoding): void
  // wrap(oldStream: NodeJS.ReadableStream): this
  push(chunk: any, encoding?: BufferEncoding): boolean{
    console.log('todo: stream.Readable.push')
    return true
  }
  _destroy(error: Error | null, callback: (error?: Error | null) => void): void{ console.log('TODO _destroy') }
  destroy(error?: Error): void{ console.log('TODO destroy') }

  /**
   * Event emitter
   * The defined events on documents including:
   * 1. close
   * 2. data
   * 3. end
   * 4. error
   * 5. pause
   * 6. readable
   * 7. resume
   */
  // addListener(event: 'close', listener: () => void): this
  // addListener(event: 'data', listener: (chunk: any) => void): this
  // addListener(event: 'end', listener: () => void): this
  // addListener(event: 'error', listener: (err: Error) => void): this
  // addListener(event: 'pause', listener: () => void): this
  // addListener(event: 'readable', listener: () => void): this
  // addListener(event: 'resume', listener: () => void): this
  // addListener(event: string | symbol, listener: (...args: any[]) => void): this {
  //   console.log('todo: stream.Readable.addListener', event)
  //   if (!this.#listeners[event]) {
  //     this.#listeners[event]  =[]
  //   }
  //   this.#listeners[event].push(listener)
  //   super.addListener(event, listener)
  //   return this
  // }

  // emit(event: 'close'): boolean
  // emit(event: 'data', chunk: any): boolean
  // emit(event: 'end'): boolean
  // emit(event: 'error', err: Error): boolean
  // emit(event: 'pause'): boolean
  // emit(event: 'readable'): boolean
  // emit(event: 'resume'): boolean
  // emit(event: string | symbol, ...args: any[]): boolean {
  //   console.log('todo: stream.Readable.emit', event)
  //   return true

  // }

  // on(event: 'close', listener: () => void): this
  // on(event: 'data', listener: (chunk: any) => void): this
  // on(event: 'end', listener: () => void): this
  // on(event: 'error', listener: (err: Error) => void): this
  // on(event: 'pause', listener: () => void): this
  // on(event: 'readable', listener: () => void): this
  // on(event: 'resume', listener: () => void): this
  // on(event: string | symbol, listener: (...args: any[]) => void): this {
  //   console.log('todo: stream.Readable.on', event)
  //   if (!this.#listeners[event]) {
  //     this.#listeners[event]  =[]
  //   }
  //   this.#listeners[event].push(listener)
  //   return this

  // }

  // once(event: 'close', listener: () => void): this
  // once(event: 'data', listener: (chunk: any) => void): this
  // once(event: 'end', listener: () => void): this
  // once(event: 'error', listener: (err: Error) => void): this
  // once(event: 'pause', listener: () => void): this
  // once(event: 'readable', listener: () => void): this
  // once(event: 'resume', listener: () => void): this
  // once(event: string | symbol, listener: (...args: any[]) => void): this {
  //   if (!this.#listeners[event]) {
  //     this.#listeners[event]  =[]
  //   }
  //   this.#listeners[event].push(listener)
  //   console.log('todo: stream.Readable.once', event)
  //   return this

  // }

  // prependListener(event: 'close', listener: () => void): this
  // prependListener(event: 'data', listener: (chunk: any) => void): this
  // prependListener(event: 'end', listener: () => void): this
  // prependListener(event: 'error', listener: (err: Error) => void): this
  // prependListener(event: 'pause', listener: () => void): this
  // prependListener(event: 'readable', listener: () => void): this
  // prependListener(event: 'resume', listener: () => void): this
  // prependListener(
  //   event: string | symbol,
  //   listener: (...args: any[]) => void
  // ): this {
  //   console.log('todo: stream.Readable.prependListener')
  //   return this

  // }

  // prependOnceListener(event: 'close', listener: () => void): this
  // prependOnceListener(event: 'data', listener: (chunk: any) => void): this
  // prependOnceListener(event: 'end', listener: () => void): this
  // prependOnceListener(event: 'error', listener: (err: Error) => void): this
  // prependOnceListener(event: 'pause', listener: () => void): this
  // prependOnceListener(event: 'readable', listener: () => void): this
  // prependOnceListener(event: 'resume', listener: () => void): this
  // prependOnceListener(
  //   event: string | symbol,
  //   listener: (...args: any[]) => void
  // ): this {
  //   console.log('todo: stream.Readable.prependOnceListener')
  //   return this

  // }

  // removeListener(event: 'close', listener: () => void): this
  // removeListener(event: 'data', listener: (chunk: any) => void): this
  // removeListener(event: 'end', listener: () => void): this
  // removeListener(event: 'error', listener: (err: Error) => void): this
  // removeListener(event: 'pause', listener: () => void): this
  // removeListener(event: 'readable', listener: () => void): this
  // removeListener(event: 'resume', listener: () => void): this
  // removeListener(
  //   event: string | symbol,
  //   listener: (...args: any[]) => void
  // ): this {
  //   console.log('todo: stream.Readable.removeListener')
  //   return this
  // }

  [Symbol.asyncIterator](): AsyncIterableIterator<any> {
    console.log('todo: stream.Readable.asyncIterator')
    return false as any
  }
}

interface WritableOptions {
    highWaterMark?: number;
    decodeStrings?: boolean;
    defaultEncoding?: BufferEncoding;
    objectMode?: boolean;
    emitClose?: boolean;
    write?(this: Writable, chunk: any, encoding: BufferEncoding, callback: (error?: Error | null) => void): void;
    writev?(this: Writable, chunks: Array<{ chunk: any, encoding: BufferEncoding }>, callback: (error?: Error | null) => void): void;
    destroy?(this: Writable, error: Error | null, callback: (error: Error | null) => void): void;
    final?(this: Writable, callback: (error?: Error | null) => void): void;
    autoDestroy?: boolean;
}

export class Writable extends Stream /* implements NodeJS.WritableStream */ {
    readonly writable: boolean = true;
    readonly writableEnded: boolean = false;
    readonly writableFinished: boolean = false;
    // readonly writableHighWaterMark: number;
    // readonly writableLength: number;
    // readonly writableObjectMode: boolean;
    // readonly writableCorked: number;
    destroyed: boolean = false
    // constructor(opts?: WritableOptions);
    _write(chunk: any, encoding: BufferEncoding, callback: (error?: Error | null) => void): void{ console.log('TODO _write') }
    _writev?(chunks: Array<{ chunk: any, encoding: BufferEncoding }>, callback: (error?: Error | null) => void): void{ console.log('TODO _writev') }
    _destroy(error: Error | null, callback: (error?: Error | null) => void): void{ console.log('TODO _destroy') }
    _final(callback: (error?: Error | null) => void): void{ console.log('TODO _final') }
    write(chunk: any, cb?: (error: Error | null | undefined) => void): boolean{
    console.log('todo: stream.Writable.write')
      return true
    }
    // write(chunk: any, encoding: BufferEncoding, cb?: (error: Error | null | undefined) => void): boolean
    // setDefaultEncoding(encoding: BufferEncoding): this;
    end(cb?: () => void): void{ console.log('TODO end') }
    // end(chunk: any, cb?: () => void): void
    // end(chunk: any, encoding: BufferEncoding, cb?: () => void): void{ console.log('TODO //') }
    cork(): void{ console.log('TODO cork') }
    uncork(): void{ console.log('TODO uncork') }
    destroy(error?: Error): void{
    console.log('todo: stream.Writable.destroy')

    }

    /**
     * Event emitter
     * The defined events on documents including:
     * 1. close
     * 2. drain
     * 3. error
     * 4. finish
     * 5. pipe
     * 6. unpipe
     */
    // addListener(event: "close", listener: () => void): this;
    // addListener(event: "drain", listener: () => void): this;
    // addListener(event: "error", listener: (err: Error) => void): this;
    // addListener(event: "finish", listener: () => void): this;
    // addListener(event: "pipe", listener: (src: Readable) => void): this;
    // addListener(event: "unpipe", listener: (src: Readable) => void): this;
    // addListener(event: string | symbol, listener: (...args: any[]) => void): this {
    // console.log('todo: stream.Writable.addListener')
    //   return this
    // }

    // emit(event: "close"): boolean;
    // emit(event: "drain"): boolean;
    // emit(event: "error", err: Error): boolean;
    // emit(event: "finish"): boolean;
    // emit(event: "pipe", src: Readable): boolean;
    // emit(event: "unpipe", src: Readable): boolean;
    // emit(event: string | symbol, ...args: any[]): boolean {
    // console.log('todo: stream.Writable.emit')
    //   return true
    // }

    // on(event: "close", listener: () => void): this;
    // on(event: "drain", listener: () => void): this;
    // on(event: "error", listener: (err: Error) => void): this;
    // on(event: "finish", listener: () => void): this;
    // on(event: "pipe", listener: (src: Readable) => void): this;
    // on(event: "unpipe", listener: (src: Readable) => void): this;
    // on(event: string | symbol, listener: (...args: any[]) => void): this {
    // console.log('todo: stream.Writable.on')
    //   return this
    // }

    // once(event: "close", listener: () => void): this;
    // once(event: "drain", listener: () => void): this;
    // once(event: "error", listener: (err: Error) => void): this;
    // once(event: "finish", listener: () => void): this;
    // once(event: "pipe", listener: (src: Readable) => void): this;
    // once(event: "unpipe", listener: (src: Readable) => void): this;
    // once(event: string | symbol, listener: (...args: any[]) => void): this {
    // console.log('todo: stream.Writable.once')
    //   return this
    // }

    // prependListener(event: "close", listener: () => void): this;
    // prependListener(event: "drain", listener: () => void): this;
    // prependListener(event: "error", listener: (err: Error) => void): this;
    // prependListener(event: "finish", listener: () => void): this;
    // prependListener(event: "pipe", listener: (src: Readable) => void): this;
    // prependListener(event: "unpipe", listener: (src: Readable) => void): this;
    // prependListener(event: string | symbol, listener: (...args: any[]) => void): this {
    // console.log('todo: stream.Writable.prependListener')
    //   return this
    // }

    // prependOnceListener(event: "close", listener: () => void): this;
    // prependOnceListener(event: "drain", listener: () => void): this;
    // prependOnceListener(event: "error", listener: (err: Error) => void): this;
    // prependOnceListener(event: "finish", listener: () => void): this;
    // prependOnceListener(event: "pipe", listener: (src: Readable) => void): this;
    // prependOnceListener(event: "unpipe", listener: (src: Readable) => void): this;
    // prependOnceListener(event: string | symbol, listener: (...args: any[]) => void): this {
    // console.log('todo: stream.Writable.prependOnceListener')
    //   return this
    // }

    // removeListener(event: "close", listener: () => void): this;
    // removeListener(event: "drain", listener: () => void): this;
    // removeListener(event: "error", listener: (err: Error) => void): this;
    // removeListener(event: "finish", listener: () => void): this;
    // removeListener(event: "pipe", listener: (src: Readable) => void): this;
    // removeListener(event: "unpipe", listener: (src: Readable) => void): this;
    // removeListener(event: string | symbol, listener: (...args: any[]) => void): this {
    // console.log('todo: stream.Writable.removeListener')
    //   return this
    // }
}


interface SocketConstructorOpts {
  fd?: number
  allowHalfOpen?: boolean
  readable?: boolean
  writable?: boolean
}

type BufferEncoding = string

export function finished(dup: Duplex, options: any) {}


// @ts-ignore
interface DuplexOptions extends ReadableOptions, WritableOptions {
  // allowHalfOpen?: boolean;
  // readableObjectMode?: boolean;
  // writableObjectMode?: boolean;
  // readableHighWaterMark?: number;
  // writableHighWaterMark?: number;
  // writableCorked?: number;
  read?(this: Duplex, size: number): void;
  write?(this: Duplex, chunk: any, encoding: BufferEncoding, callback: (error?: Error | null) => void): void;
  // writev?(this: Duplex, chunks: Array<{ chunk: any, encoding: BufferEncoding }>, callback: (error?: Error | null) => void): void;
  // final?(this: Duplex, callback: (error?: Error | null) => void): void;
  destroy?(this: Duplex, error: Error | null, callback: (error: Error | null) => void): void;
}

// Note: Duplex extends both Readable and Writable.
// @ts-ignore
export  class Duplex extends Readable implements Writable {
  readonly writable: boolean = true;
  readonly writableEnded: boolean = false;
  readonly writableFinished: boolean = false;
  // readonly writableHighWaterMark: number = 1;
  // readonly writableLength: number = 1;
  // readonly writableObjectMode: boolean = false;
  // readonly writableCorked: number = 1;
  constructor(opts?: DuplexOptions) {
    super()
  }
  _write(chunk: any, encoding: BufferEncoding, callback: (error?: Error | null) => void): void {
    console.log('todo: stream.duplex._write')
  }
  _writev?(chunks: Array<{ chunk: any, encoding: BufferEncoding }>, callback: (error?: Error | null) => void): void {
    console.log('todo: stream.duplex._writev')
  }
  _destroy(error: Error | null, callback: (error: Error | null) => void): void {
    console.log('todo: stream.duplex._destroy')
  }
  _final(callback: (error?: Error | null) => void): void {
    console.log('todo: stream.duplex._final')
  }
  write(chunk: any, cb?: (error: Error | null | undefined) => void): boolean {
    console.log('todo: stream.duplex.write')
    return true
  }
  setDefaultEncoding(encoding: BufferEncoding): this {
    console.log('todo: stream.duplex.setDefaultEncoding')
    return this
  }
  end(cb?: () => void): void {
    console.log('todo: stream.duplex.end')

  }
  cork(): void {}
  uncork(): void {}


  // Wirtable?
  // [Symbol.asyncIterator]() {
  //   return {
  //     next() {
  //       return new Promise<{ value: any }>(res => {})
  //     },
  //   }
  // }
}
applyMixins (Duplex, [Writable]);

function applyMixins(derivedCtor: any, baseCtors: any[]) {
    baseCtors.forEach(baseCtor => {
        Object.getOwnPropertyNames(baseCtor.prototype).forEach(name => {
             if (name !== 'constructor') {
                derivedCtor.prototype[name] = baseCtor.prototype[name];
            }
        });
    }); 
}