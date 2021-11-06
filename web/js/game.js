
export function promoteCustom(o) {
  for (let x in o.custom) {
    o[x] = o.custom[x]
  }
  delete o.custom
  return o
}

function defaultProcessUpdate(u) {
  promoteCustom(u)

  u.players = u.players || []

  for (let pln in u.players) {
    let pl = u.players[pln]
    pl.number = pln
    promoteCustom(pl)
  }

  return u
}

function defaultProcessTurn(t) {
  promoteCustom(t)
  t.can = t.can  || []
  t.must = t.must || []
  return t
}

export function connect(listener, args) {
  let netState = {
    ws: null,

    reqNo: 0,
    reqs: new Map(),

    send() {},
    doRequest() {},
  }

  let q = ''
  for (let k in args) {
    q += k + '=' + args[k] + '&'
  }

  const conn = new WebSocket(`ws://${location.host}/ws?${q}`, 'comms')

  conn.onclose = e => {
    console.log(`WebSocket Disconnected code: ${e.code}, reason: ${e.reason}`)
    netState.ws = null
    listener.onDisconnect()
    if (e.code !== 1001) {
      setTimeout(() => {
        connect(listener, args)
      }, 5000)
    }
  }

  conn.onopen = _e => {
    netState.ws = conn
    listener.onConnect()

    let send = (type, data) => {
      let msg = {
        Head: type,
        Data: data
      }

      console.log('tx', msg)
      let jtext = JSON.stringify(msg)
      conn.send(jtext)
    }

    let request = (rtype, body, then) => {
      let rn = '' + netState.reqNo++
      let mtype = 'request:' + rn + ':' + rtype
      netState.reqs.set(rn, then)
      send(mtype, body)
    }

    netState.send = send
    netState.doRequest = request
  }

  conn.onmessage = e => {
    if (typeof e.data !== "string") {
      console.error("unexpected message type", typeof e.data)
      return
    }
    let msg = JSON.parse(e.data)
    console.log('rx', msg)
    if (msg.head === 'update') {
      setTimeout(() => listener.onUpdate(msg.data), 0)
    } else if (msg.head === 'turn') {
      setTimeout(() => listener.onTurn(msg.data), 0)
    } else if (msg.head === 'text') {
      setTimeout(() => listener.onText(msg.data), 0)
    } else if (msg.head.startsWith('response:')) {
      let rn = msg.head.substring(9)
      let then = netState.reqs.get(rn)
      netState.reqs.delete(rn)

      let res = msg.data
      // XXX - nothing says these fields must exist
      setTimeout(() => then(res.error, res), 0)
    }
  }

  return netState
}

export function newUI(data, gameId, name, colour, processUpdate, processTurn) {
  processTurn = processTurn || (e => e)
  processUpdate = processUpdate || (e => e)
  let state = {
    data: data,
    gameId: gameId,
    status: null,
    me: {
      name: name,
      colour: colour,
    },
    playing: null,
    players: {}
  }

  let hasCommand = (list, pattern) => {
    for (let can of list) {
      if (can === pattern) {
        return true
      }
    }
    return false
  }

  let newTurn = t => {
    t.can = t.can || []
    t.must = t.must || []
    t.hasCan = pattern => {
      return hasCommand(t.can, pattern)
    }
    t.hasMust = pattern => {
      return hasCommand(t.must, pattern)
    }
    return t
  }

  let turn = newTurn({})

  let upStream = {
    send(cmd) {
      sendCommand(cmd)
    },
    play(cmd, options, cb) {
      if (options) {
        cmd += ':' + options
      }
      netState.doRequest('play', { command: cmd }, cb)
    }
  }

  let components = []

  let addComponent = (f) => {
    components.push(f(data, upStream))
  }

  let sendCommand = (cmd) => {
    console.log('command', cmd)
    for (let c of components) {
      c.onCommand && c.onCommand(cmd)
    }
  }

  let dumpState = () => {
    console.log(state)
  }

  let onConnect = () => {
    document.body.setAttribute('connected', true)
  }

  let onDisconnect = () => {
    document.body.setAttribute('connected', false)
  }

  let onUpdate = u => {
    u = processUpdate(defaultProcessUpdate(u))

    state.status = u.status
    state.winner = u.winner
    state.playing = u.playing
    for (let pl of u.players) {
      state.players[pl.name] = pl
      if (pl.name == state.me.name) {
        state.me = pl
      }
    }
    if (state.playing != state.me.name) {
      turn = newTurn({})
      for (let c of components) {
        c.onTurn && c.onTurn(turn)
      }
    }
    for (let c of components) {
      c.onUpdate && c.onUpdate(state)
    }
    for (let n of u.news) {
      sendCommand({ do: 'log', msg: n })
    }
  }

  let onTurn = t => {
    t = processTurn(defaultProcessTurn(t))
    turn = newTurn(t)
    for (let c of components) {
      c.onTurn && c.onTurn(turn)
    }
  }

  let onText = t => {
    sendCommand({ do: 'log', msg: t })
  }

  return {
    addComponent,
    sendCommand,
    dumpState,
    onConnect,
    onDisconnect,
    onUpdate,
    onTurn,
    onText
  }
}
