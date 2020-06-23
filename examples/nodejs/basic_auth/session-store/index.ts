export class SessionStore {
  store: any;

  constructor(store: any) {
    this.store = store
  }

  public load(headers: { [key:string]:string }) {
    return this.store.get(headers.sessionId)
  }

  public create(sessionId: string, session: any) {
    this.store.set(sessionId, session)
  }

  public clear(sessionId: string) {
    this.store.destroy(sessionId)
  }
}