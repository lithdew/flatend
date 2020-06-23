import { Context } from "../context";
import { Memory } from "./memory";

export class SessionStore {
  store: Memory;
  mode: string;

  constructor(mode: string) {
    this.mode = mode
    switch(mode) { 
      case "memory": {
        this.store = new Memory();
        break; 
      }
      default: {
        this.store = new Memory();
        break; 
      } 
    }
  }

  public load(sessionid: string) {
    return sessionid? Object.assign(this.store.get(sessionid), {id: sessionid}) : null
  }

  public create(sessionid: string, session: any) {
    this.store.set(sessionid, session)
  }

  public clear(sessionid: string) {
    this.store.destroy(sessionid)
  }
}