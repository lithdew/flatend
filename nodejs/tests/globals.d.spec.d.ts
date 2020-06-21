/// <reference types="chai" />
declare module Chai {
  interface Assertion {
    equalBytes(expected: ArrayBufferLike): Chai.Equal;
  }
}
