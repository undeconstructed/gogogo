
export function newUI(data, gameId, name, colour) {
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
