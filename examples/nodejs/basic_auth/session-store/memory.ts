export class Memory {
  sessions: { [key: string]: any }

  constructor(
    sessions: { [key: string]: any } = Object.create(null)
  ) {
    this.sessions = Object.create(null)
  }

  public getSession(sessionId: string) {
    var sess = this.sessions[sessionId]

    if (!sess) {
      return
    }

    // parse
    sess = JSON.parse(sess)

    if (sess.cookie) {
      var expires = typeof sess.cookie.expires === 'string'
        ? new Date(sess.cookie.expires)
        : sess.cookie.expires

      // destroy expired session
      if (expires && expires <= Date.now()) {
        delete this.sessions[sessionId]
        return
      }
    }

    return sess
  }

  public all() {
    var sessionIds = Object.keys(this.sessions)
    var sessions = Object.create(null)

    for (var i = 0; i < sessionIds.length; i++) {
      var sessionId = sessionIds[i]
      var session = this.getSession(sessionId)

      if (session) {
        sessions[sessionId] = session;
      }
    }

    return sessions
  }

  public set(sessionId: string, session: any) {
    this.sessions[sessionId] = JSON.stringify(session)
  }

  public get(sessionId: string) {
    return this.getSession(sessionId)
  }

  public destroy(sessionId: string) {
    delete this.sessions[sessionId]
  }
}