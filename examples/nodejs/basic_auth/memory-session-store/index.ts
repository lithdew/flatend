import { Context } from "../context";
import { Store } from "./store";

export class MemorySessionStore {
  sessionid?: string | null;
  store?: any;

  constructor() {
    this.store = new Store();
  }

  public load(sessionid: string): any {
    return this.sessionid? Object.assign(this.store.get(sessionid), {id: sessionid}) : null
  }

  public create(sessionid: string, session: any) {
    this.sessionid = sessionid
    this.store.set(sessionid, session)
  }

  public clear(sessionid: string) {
    this.sessionid = null
    this.store.destroy(sessionid)
  }
}