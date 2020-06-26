export class SessionStore {
  store: any;

  constructor(store: any) {
    this.store = store
  }

  public load(headers: { [key:string]:string }) {
    if(!headers.sessionId) {
      return null
    } else { 
      return this.store.get(headers.sessionId)
    }
  }

  public create(sessionId: string, payload: any) {
    var maxAge = 86400;
    var oneDay = 86400;
    var now = new Date().getTime();
    var expiry = maxAge ? now + maxAge : now + oneDay;
    this.store.set(sessionId, Object.assign({
      sid: sessionId,
      expiry: expiry
    }, payload))
  }

  public clear(sessionId: string) {
    this.store.destroy(sessionId)
  }
}