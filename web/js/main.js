
// utils

function select(parent, selector) {
  let e = parent.querySelector(selector)
  if (e == null) {
    console.log('null', parent, selector)
    return
  }
  return e
}

// state

let state = {
  data: null,
  ws: null,

  reqNo: 0,
  reqs: new Map(),

  squares: [],
  trackMarks: new Map(),
  mapMarks: new Map(),

  players: new Map(),
  playing: null,
  turn: null,
}

// game utils

function makeByLine(modes) {
  let ms = []
  for (let m of modes) {
    ms.push(state.data.modes[m])
  }
  return ms.join('/')
}

function splitDotId(s) {
  let ss = s.split(',')
  return [parseInt(ss[0]), parseInt(ss[1])]
}

// comms

function connect() {
  if (state.ws) return

  const conn = new WebSocket(`ws://${location.host}/ws?game=${state.gameId}&name=${state.name}&colour=${state.colour}`, 'comms')

  conn.onclose = ev => {
    console.log(`WebSocket Disconnected code: ${ev.code}, reason: ${ev.reason}`)
    document.body.setAttribute('connected', false)
    state.ws = null
    if (ev.code !== 1001) {
      setTimeout(connect, 5000)
    }
  }

  conn.onopen = ev => {
    document.body.setAttribute('connected', true)
    state.ws = conn
  }

  conn.onmessage = ev => {
    if (typeof ev.data !== "string") {
      console.error("unexpected message type", typeof ev.data)
      return
    }
    let msg = JSON.parse(ev.data)
    console.log('rx', msg)
    if (msg.head === 'update') {
      receiveUpdate(msg.data)
    } else if (msg.head === 'turn') {
      receiveTurn(msg.data)
    } else if (msg.head === 'text') {
      log(msg.data)
    } else if (msg.head.startsWith('response:')) {
      let rn = msg.head.substring(9)
      let then = state.reqs.get(rn)
      state.reqs.delete(rn)

      let res = msg.data
      // XXX - nothing says these fields must exist
      then(res.error, res)
    }
  }
}

function send(type, data) {
  if (!state.ws) {
    alert('not connected')
    return
  }

  let msg = {
    Head: type,
    Data: data
  }

  console.log('tx', msg)
  let jtext = JSON.stringify(msg)
  state.ws.send(jtext)
}

// receiving data

function promoteCustom(o) {
  for (let x in o.custom) {
    o[x] = o.custom[x]
  }
  delete o.custom
  return o
}

function receiveUpdate(st) {
  promoteCustom(st)

  if (!st.playing) {
    document.body.setAttribute('started', false)
  } else {
    document.body.setAttribute('started', true)
  }

  if (state.playing != st.playing) {
    // turn has moved on
    state.playing = st.playing
  }

  for (let pl of st.players || {}) {
    promoteCustom(pl)

    // tidy up data
    pl.souvenirs = pl.souvenirs || []
    pl.lucks = pl.lucks || []
    pl.money = pl.money || {}
    for (let k in pl.money) {
      if (pl.money[k] == 0) {
        delete pl.money[k]
      }
    }

    let prev = state.players.get(pl.name) || {}

    if (pl.name == state.name) {
      // this is us
      receiveLucks(pl.lucks)
      receiveTicket(pl.ticket)
      receiveSouvenirs(pl.souvenirs)
      receiveMoney(pl.money)
    }

    if (prev.square != pl.square) {
      markOnTrack(pl.colour, pl.square)
    }
    if (prev.dot != pl.dot) {
      markOnMap(pl.colour, pl.dot)
    }

    if (state.playing == pl.name) {
      // focus on active plauer
      scrollMapTo(pl.dot)
      scrollTrackTo(pl.square)
    }

    state.players.set(pl.name, pl)
  }

  let md = select(document, '.aboutgame .players > div')
  md.replaceChildren()
  for (let pl of state.players.values()) {
    let div = document.createElement('div')
    div.classList.add('aplayer')
    let colSpan = document.createElement('span')
    colSpan.classList.add('colour')
    colSpan.style.backgroundColor = pl.colour || 'transparent'
    div.append(colSpan)
    let nameSpan = document.createElement('span')
    nameSpan.classList.add('name')
    nameSpan.append(pl.name)
    div.append(nameSpan)
    md.append(div)
  }

  for (let n of st.news) {
    log(n)
  }

  if (st.playing && st.playing != state.name) {
    // XXX - is not my turn, update the status bar - shouldn't use a fake turn
    receiveTurn({
      player: st.playing
    })
  }
}

function receiveLucks(lucks) {
  let stack = select(document, '.lucks')
  stack.replaceChildren()

  if (lucks.length == 0) {
    document.body.setAttribute('hasluck', false)
    closeLucks()
  } else {
    document.body.setAttribute('hasluck', true)

    let tmpl = select(document, '#lucktemplate').content.firstElementChild

    for (let luckId of lucks) {
      let luckData = state.data.lucks[luckId]
      let div = tmpl.cloneNode(true)
      select(div, '.body').textContent = luckData.name
      select(div, 'button').addEventListener('click', e => {
        e.stopPropagation()
        useLuck(luckId)
        closeLucks()
      })
      stack.append(div)
    }

    if (stack.classList.contains('open')) {
      openLucks()
    }
  }
}

function receiveTicket(ticket) {
  if (!ticket) {
    document.body.setAttribute('ticket', false)
  } else {
    let div = select(document, '.ticket')
    select(div, '.by > span').textContent = makeByLine(ticket.by)
    let from = state.data.places[ticket.from].name
    select(div, '.from > span').textContent = from
    let to = state.data.places[ticket.to].name
    select(div, '.to > span').textContent = to
    let currency = state.data.currencies[ticket.currency].name
    select(div, '.fare > span').textContent = `${ticket.fare} ${currency}`
    document.body.setAttribute('ticket', true)
  }
}

function receiveSouvenirs(souvenirs) {
  let stack = select(document, '.souvenirs')
  stack.replaceChildren()

  if (souvenirs.length == 0) {
    document.body.setAttribute('hassouvenir', false)
    closeSouvenirs()
  } else {
    document.body.setAttribute('hassouvenir', true)

    let tmpl = select(document, '#souvenirtemplate').content.firstElementChild

    for (let placeId of souvenirs) {
      let place = state.data.places[placeId]
      let currency = state.data.currencies[place.currency]
      let div = tmpl.cloneNode(true)
      select(div, '.where').textContent = 'Souvenir from ' + place.name
      let price = state.data.settings.souvenirPrice * currency.rate
      select(div, '.price').textContent = '' + price + ' ' + currency.name
      for (let bar of div.querySelectorAll('.bar')) {
        bar.style.backgroundColor = currency.colour
      }
      select(div, 'button').addEventListener('click', e => {
        e.stopPropagation()
        declareSouvenir(placeId)
        closeSouvenirs()
      })
      stack.append(div)
    }

    if (stack.classList.contains('open')) {
      openSouvenirs()
    }
  }
}

function receiveMoney(money) {
  let stack = select(document, '.money')
  stack.replaceChildren()

  if (Object.keys(money).length == 0) {
    document.body.setAttribute('hasmoney', false)
    closeMoney()
  } else {
    document.body.setAttribute('hasmoney', true)
    let tmpl = select(document, '#moneytemplate').content.firstElementChild

    for (let cId in money) {
      let amount = money[cId]
      if (amount) {
        let currency = state.data.currencies[cId]
        let div = tmpl.cloneNode(true)
        select(div, '.head').textContent = cId // currency.name
        select(div, '.body').textContent = '' + money[cId]
        div.style.backgroundColor = currency.colour
        // if (cId === 'ye') {
        //   select(div, '.inner').style.backgroundImage = 'url(img/money_ye.svg)'
        // }
        select(div, 'button').addEventListener('click', e => {
          e.stopPropagation()
          changeMoney(cId)
          closeMoney()
        })
        stack.append(div)
      }
    }

    if (stack.classList.contains('open')) {
      openMoney()
    }
  }
}

function receiveTurn(st) {
  promoteCustom(st)

  state.turn = st
  state.turn.can = state.turn.can || []
  state.turn.must = state.turn.must || []

  document.body.setAttribute('canluck', false)
  document.body.setAttribute('canchangemoney', false)
  document.body.setAttribute('mustdeclare', false)
  hideDiceButton()
  hideStopButton()
  hideSleepButton()

  for (let cmd of state.turn.can) {
    if (cmd === 'useluck:*') {
      document.body.setAttribute('canluck', true)
    } else if (cmd.startsWith('changemoney:')) {
      document.body.setAttribute('canchangemoney', true)
    } else if (cmd === 'dicemove') {
      showDiceButton()
    } else if (cmd === 'stop') {
      showStopButton()
    } else if (cmd === 'end') {
      showSleepButton()
    }
  }
  for (let cmd of state.turn.must) {
    if (cmd === 'declare:*') {
      document.body.setAttribute('mustdeclare', true)
    }
  }

  let player = state.players.get(st.player)
  document.body.setAttribute('ontrack', !st.onmap)

  let s = select(document, '.aboutgame')

  let sc = select(s, '.colour')
  sc.style.backgroundColor = player.colour
  let sn = select(s, '.name')
  sn.textContent = player.name

  makeButtons()
}

function markOnTrack(colour, square) {
  let prev = state.trackMarks.get(colour)
  if (prev) {
    prev.remove()
  }

  if (!colour) {
    return
  }

  let mark = document.createElement('div')
  mark.classList.add('mark')
  mark.style.backgroundColor = colour
  let squareDiv = state.squares[square]

  state.trackMarks.set(colour, mark)
  select(squareDiv, '.sitting').append(mark)
}

function scrollTrackTo(square) {
  let squareDiv = state.squares[square]
  squareDiv.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

function markOnMap(colour, dot) {
  let prev = state.mapMarks.get(colour)
  if (prev) {
    prev.remove()
  }

  if (!colour) {
    return
  }

  let svg = select(document, '.map > object').contentDocument
  let layer = select(svg, '#dotslayer')

  let [x, y] = splitDotId(dot)

  let marker = select(svg, '#playerring')
  let nmarker = marker.cloneNode()
  nmarker.id = 'player-' + colour
  nmarker.setAttributeNS(null, 'cx', x);
  nmarker.setAttributeNS(null, 'cy', y);
  nmarker.style.stroke = colour

  state.mapMarks.set(colour, nmarker)
  layer.append(nmarker)
}

function scrollMapTo(dot) {
  let [x, y] = splitDotId(dot)

  let scroller = select(document, '.map')
  let scrollee = scroller.firstElementChild
  let sLeft = (x/1000)*scrollee.offsetWidth-scroller.offsetWidth/2+scrollee.offsetLeft
  let sTop = (y/700)*scrollee.offsetHeight-scroller.offsetHeight/2+scrollee.offsetTop
  scroller.scrollTo({ top: sTop, left: sLeft, behavior: 'smooth' })
}

function makeButtons() {
  let buttonBox = select(document, '.actions')
  buttonBox.replaceChildren()

  // {
  //   let button = document.createElement('button')
  //   button.append('say')
  //   button.addEventListener('click', doSay)
  //   buttonBox.append(button)
  // }

  if (state.turn) {
    makePlayButtons(buttonBox, state.turn.can, 'can')
    makePlayButtons(buttonBox, state.turn.must, 'must')
  }

  // emergency button
  // let button = document.createElement('button')
  // button.classList.add('text')
  // button.append('??')
  // button.addEventListener('click', e => {
  //   let cmd = prompt('??')
  //   if (!cmd) { return; }
  //   doRequest('play', { command: cmd }, console.log)
  // })
  // buttonBox.append(button)

  showButtons()
}

function showButtons() {
  let buttonBox = select(document, '.actions')
  buttonBox.classList.add('open')
}

function hideButtons() {
  let buttonBox = select(document, '.actions')
  buttonBox.classList.remove('open')
}

function makePlayButtons(tgt, actions, clazz) {
  for (let a of actions || []) {
    let parts = a.split(":")

    let cmd = parts[0]
    let action = state.data.actions[cmd]

    let button = document.createElement('button')
    button.classList.add(clazz)

    let cb = null

    if (cmd === 'useluck') {
      // can do this with the cards
      continue
    } else if (cmd === 'buyticket') {
      // use the price list, but you have to notice that you are allowed
      // let placeId = parts[1]
      // showPrices(placeId)
      continue
    } else if (cmd === 'changemoney') {
      // can do this with the notes
      continue
    } else if (cmd === 'declare') {
      // can do this with the cards
      continue
    } else if (cmd === 'dicemove' || cmd === 'stop' || cmd === 'end') {
      // there's a big button somewhere
      continue
    } else if (cmd === 'dicemove') {
      // button.classList.add('dice')
      // cb = r => showLogLine('you rolled a ' + r.message)
    } else if (cmd === 'gamble') {
      button.classList.add('dice')
      cb = r => showLogLine('you gambled a ' + r.message)
    } else if (cmd === 'takeluck') {
      setupForTakeLuck()
      continue
    } else if (cmd === 'takerisk') {
      setupForTakeRisk()
      continue
    } else if (cmd === 'obeyrisk') {
      let cardId = parseInt(parts[1])
      setupForObeyRisk(cardId)
      continue
    } else if (cmd === 'ignorerisk') {
      // only happens when obey is also enabled
      continue
    } else if (cmd === 'buysouvenir') {
      button.classList.add('buysouvenir')
      // we know this command is complete, so no prompt
      cmd = a
      action = null
      cb = r => showLogLine('you have bought a ' + r.message)
    } else if (cmd === 'end') {
      button.classList.add('end')
    } else {
      button.classList.add('text')
      button.append(cmd)
    }

    button.addEventListener('click', e => {
      doPlay(cmd, action, cb)
    })
    tgt.append(button)
  }
}

// actions

function doSay() {
  let msg = prompt('Say what?')
  if (!msg) return
  send('text', msg)
}

function doRequest(rtype, body, then) {
  let rn = '' + state.reqNo++
  let mtype = 'request:' + rn + ':' + rtype
  state.reqs.set(rn, then)
  send(mtype, body)
}

function doStart() {
  doRequest('start', null, (e, r) => {
    if (e) {
      alert(e.message); return
    }
    // log(r)
  })
}

function doPlay(cmd, action, cb) {
  let options = null
  if (action && action.help) {
    options = prompt(`${cmd} ${action.help}`)
    if (!options) return
  }

  hideButtons()

  let cb0 = (e, r) => {
    if (e) {
      alert(e.message)
      showButtons()
      return
    }
    cb && cb(r)
  }

  if (options) {
    cmd += ':' + options
  }

  doRequest('play', { command: cmd }, cb0)
}

function useLuck(id) {
  let options = prompt('options (or none)')

  let cb = (e, r) => {
    if (e) { alert(e.message); return; }
  }

  doRequest('play', { command: 'useluck:'+id+':'+options }, cb)
}

function changeMoney(from) {
  let to = ''
  // XXX - HORRIBLE!!!
  for (let can of state.turn.can) {
    if (can.startsWith('changemoney:')) {
      to = can.substring(14, 16)
      break
    }
  }

  let options = prompt('how much?')

  let cb = (e, r) => {
    if (e) { alert(e.message); return; }
  }

  doRequest('play', { command: `changemoney:${from}:${to}:${options}` }, cb)
}

function declareSouvenir(placeId) {
  let cb = (e, r) => {
    if (e) { alert(e.message); return; }
  }

  doRequest('play', { command: 'declare:'+placeId }, cb)
}

function doDiceMove() {
  let cb = (e, r) => {
    if (e) { alert(e.message); return; }
    showLogLine('you rolled a ' + r.message)
  }

  doRequest('play', { command: 'dicemove' }, cb)
}

function doStop() {
  let cb = (e, r) => {
    if (e) { alert(e.message); return; }
  }

  doRequest('play', { command: 'stop' }, cb)
}

function doEnd() {
  let cb = (e, r) => {
    if (e) { alert(e.message); return; }
  }

  doRequest('play', { command: 'end' }, cb)
}

// ui manipulation

function makeLuckStack() {
  let stack = select(document, '.lucks')
  stack.addEventListener('click', e => {
    if (stack.classList.contains('stashed')) {
      openLucks()
    } else {
      closeLucks()
    }
  })
}

function openLucks() {
  let stack = select(document, '.lucks')

  stack.classList.remove('stashed')
  stack.classList.add('open')

  let turn = .05
  let howMany = stack.querySelectorAll('.luckcard').length
  let totalTurn = howMany * turn

  let n = -(totalTurn/2)
  for (let card of stack.querySelectorAll('.luckcard')) {
    card.style.rotate = n + 'turn'
    n += turn
  }
}

function closeLucks() {
  let stack = select(document, '.lucks')

  stack.classList.remove('open')
  stack.classList.add('stashed')

  for (let card of stack.querySelectorAll('.luckcard')) {
    card.style.rotate = 'unset'
  }
}

function makeSouvenirPile() {
  let stack = select(document, '.souvenirs')
  stack.addEventListener('click', e => {
    if (stack.classList.contains('stashed')) {
      openSouvenirs()
    } else {
      closeSouvenirs()
    }
  })
}

function openSouvenirs() {
  let stack = select(document, '.souvenirs')

  stack.classList.remove('stashed')
  stack.classList.add('open')

  let move = -3

  let n = 0
  for (let card of stack.querySelectorAll('.souvenircard')) {
    card.style.left = n + 'rem'
    n += move
  }
}

function closeSouvenirs() {
  let stack = select(document, '.souvenirs')

  stack.classList.remove('open')
  stack.classList.add('stashed')

  for (let card of stack.querySelectorAll('.souvenircard')) {
    card.style.left = 0
  }
}

function makeMoneyPile() {
  let stack = select(document, '.money')
  stack.addEventListener('click', e => {
    if (stack.classList.contains('stashed')) {
      openMoney()
    } else {
      closeMoney()
    }
  })
}

function openMoney() {
  let stack = select(document, '.money')

  stack.classList.remove('stashed')
  stack.classList.add('open')

  let turn = .05
  let howMany = stack.querySelectorAll('.banknote').length
  let totalTurn = howMany * turn

  let n = -(totalTurn/2)
  for (let note of stack.querySelectorAll('.banknote')) {
    note.style.rotate = n + 'turn'
    n += turn
  }
}

function closeMoney() {
  let stack = select(document, '.money')

  stack.classList.remove('open')
  stack.classList.add('stashed')

  for (let card of stack.querySelectorAll('.banknote')) {
    card.style.rotate = 'unset'
  }
}

function setupForTakeLuck() {
  let div = select(document, '.showluck')
  div.classList.add('blank')
  div.classList.add('back')
  div.classList.add('open')

  let button = select(div, '.button')
  button.addEventListener('click', e => {
    let cb = (e, r) => {
      if (e) {
        alert(e.message)
        setupForTakeLuck()
        return
      }
      showLuckCard(r.message)
    }
    doRequest('play', { command: 'takeluck' }, cb)
  }, { once: true })
}

function showLuckCard(cardId) {
  let card = state.data.lucks[cardId]
  if (!card) {
    card = {
      name: 'there are only so many cards'
    }
  }

  let div = select(document, '.showluck')
  div.classList.remove('back')
  div.classList.add('middle')

  setTimeout(() => {
    div.classList.remove('middle')
    div.classList.remove('blank')
    select(div, '.card .body').textContent = card.name
  }, 500)

  div.addEventListener('click', e => {
    hideLuck()
  }, { once: true })
}

function hideLuck() {
  let div = select(document, '.showluck')
  div.classList.remove('open')
}

function setupForTakeRisk() {
  let div = select(document, '.showrisk')
  div.classList.add('blank')
  div.classList.add('back')
  div.classList.add('open')

  let button = select(div, '.button')
  button.addEventListener('click', e => {
    let cb = (e, r) => {
      if (e) {
        alert(e.message)
        setupForTakeRisk()
        return
      }
      showRiskCard(r.message)
    }
    doRequest('play', { command: 'takerisk' }, cb)
  }, { once: true })
}

function setupForObeyRisk(cardId) {
  // XXX - this will probably interrupt the turn animation
  let div = select(document, '.showrisk')
  div.classList.add('open')
  div.classList.remove('back')
  div.classList.remove('middle')
  div.classList.remove('blank')

  let card = state.data.risks[cardId]
  select(div, '.card .body').textContent = card.name

  let foot = select(div, '.foot')
  foot.replaceChildren()

  let buttons = []

  if (state.turn.can.includes('ignorerisk:'+cardId)) {
    let ignoreButton = document.createElement('button')
    ignoreButton.append('[ignore]')
    ignoreButton.addEventListener('click', e => {
      hideRisk()
      let cb = (e, r) => {
        if (e) {
          setupForObeyRisk(cardId)
          alert(e.message)
          return
        }
      }
      doRequest('play', { command: 'ignorerisk:'+cardId }, cb)
    }, { once: true })
    buttons.push(ignoreButton)
  }

  if (state.turn.must.includes('obeyrisk:'+cardId)) {
    let obeyButton = document.createElement('button')
    obeyButton.append('[obey]')
    obeyButton.addEventListener('click', e => {
      hideRisk()
      let cb = (e, r) => {
        if (e) {
          setupForObeyRisk(cardId)
          alert(e.message)
          return
        }
      }
      doRequest('play', { command: 'obeyrisk:'+cardId }, cb)
    }, { once: true })
    buttons.push(obeyButton)
  }

  foot.replaceChildren(...buttons)
}

function showRiskCard(cardId) {
  let card = state.data.risks[cardId]

  let div = select(document, '.showrisk')
  let foot = select(div, '.foot')
  foot.replaceChildren()
  div.classList.remove('back')
  div.classList.add('middle')

  setTimeout(() => {
    div.classList.remove('middle')
    div.classList.remove('blank')
    select(div, '.card .body').textContent = card.name

    // will be followed by update, and the obey button will appear
    div.addEventListener('click', e => {
      if (!state.turn.must.includes('obeyrisk:'+cardId)) {
        // need to disable this if obey is enabled, but this is un ugly way to do it
        hideRisk()
      }
    }, { once: true })
  }, 500)
}

function hideRisk() {
  let div = select(document, '.showrisk')
  div.classList.remove('open')
}

function showPrices(placeId) {
  let ele = select(document, '.prices')
  let tbody = select(ele, 'tbody')
  tbody.replaceChildren()

  let place = state.data.places[placeId]
  let currency = state.data.currencies[place.currency]

  let stRate = state.data.currencies['st'].rate
  let loRate = currency.rate

  let linen = 0
  for (let r in place.routes) {
    let tr = document.createElement('tr')
    let th = document.createElement('th')
    if (linen == 0) {
      th.classList.add('place')
      th.textContent = place.name
    } else if (linen == 1) {
      th.textContent = `(${currency.name})`
    }
    tr.append(th)
    linen++

    let ss = r.split(':')
    let destId = ss[0]
    let dest = state.data.places[destId].name
    let modeId = ss[1]
    let mode = makeByLine(modeId)
    let fare = place.routes[r]

    let td1 = document.createElement('td')
    td1.classList.add('place')
    td1.append(dest)
    tr.append(td1)

    let td2 = document.createElement('td')
    td2.append(mode)
    tr.append(td2)

    let td3 = document.createElement('td')
    td3.classList.add('fare')
    td3.append(`£${fare*stRate}`)
    tr.append(td3)
    let td4 = document.createElement('td')
    td4.classList.add('fare')
    td4.append(`${fare*loRate}`)
    tr.append(td4)

    tr.addEventListener('click', e => {
      let cb = r => showLogLine('you have bought a ticket')

      doPlay(`buyticket:${placeId}:${destId}:${modeId}`, null, cb)
    })

    tbody.append(tr)
  }

  ele.classList.add('open')
}

function hidePrices() {
  let ele = select(document, '.prices')
  ele.classList.remove('open')
}

function makeStopButton() {
  let div = select(document, '.stopbutton')
  div.addEventListener('click', e => {
    doStop()
    hideStopButton()
  })
}

function showStopButton() {
  let div = select(document, '.stopbutton')
  div.classList.add('open')
}

function hideStopButton() {
  let div = select(document, '.stopbutton')
  div.classList.remove('open')
}

function makeSleepButton() {
  let div = select(document, '.sleepbutton')
  div.addEventListener('click', e => {
    doEnd()
    hideSleepButton()
  })
}

function showSleepButton() {
  let div = select(document, '.sleepbutton')
  div.classList.add('open')
}

function hideSleepButton() {
  let div = select(document, '.sleepbutton')
  div.classList.remove('open')
}

function makeDiceButton() {
  let div = select(document, '.dicebutton')
  div.addEventListener('click', e => {
    doDiceMove()
    hideDiceButton()
  })
}

function showDiceButton() {
  let div = select(document, '.dicebutton')
  div.classList.add('open')
}

function hideDiceButton() {
  let div = select(document, '.dicebutton')
  div.classList.remove('open')
}
// game setup

function fixupData(indata) {
  for (let dotId in indata.dots) {
    let dot = indata.dots[dotId]
    if (dot.place) {
      let place = indata.places[dot.place]
      dot.terminal = true
      if (place.city) {
        dot.city = true
      }
    }
  }

  return indata
}

function setup(inData, gameId, name, colour) {
  state.data = fixupData(inData)
  state.gameId = gameId
  state.name = name
  state.colour = colour

  let s = select(document, '.aboutme')
  let sc = select(s, '.colour')
  sc.style.backgroundColor = state.colour
  let sn = select(s, '.name')
  sn.textContent = state.name

  let startButton = select(document, '#startbutton')
  startButton.addEventListener('click', doStart)

  makeSquares(state.data)
  plot(state.data)
  scrollingMap()
  makeButtons()
  makeLuckStack()
  makeSouvenirPile()
  makeMoneyPile()

  makeStopButton()
  makeSleepButton()
  makeDiceButton()

  // select(document, '.showluck').addEventListener('click', hideLuck)
  // select(document, '.showrisk').addEventListener('click', hideRisk)

  select(document, '.prices').addEventListener('click', hidePrices)

  connect()
}

function makeSquares(data) {
  let z = 1000

  let area = select(document, '.squares')
  for (let squareId in data.squares) {
    let square = data.squares[squareId]

    let el = document.createElement('div')
    el.style.zIndex = z--

    let background = 'squarex.svg'
    if (square.type == 'customs1' || square.type == 'customs2' || square.type === 'luck' || square.type === 'hospital' || square.type === 'hotel') {
      // some squares are yellow
      // el.style.backgroundColor = '#ddb700'
      background = 'squarey.svg'
    }

    // if using background
    el.style.backgroundImage = `url(img/squarez.svg), url(img/${square.type}.svg), url(img/${background})`

    // if using images
    // let i = document.createElement('img')
    // i.src = square.type+'.svg'
    // el.append(i)

    // if there is no image:
    // for (let s of square.name.split(' - ')) {
    //   let d = document.createElement('div')
    //   d.append(s)
    //   el.append(d)
    // }

    let sittingRoom = document.createElement('div')
    sittingRoom.classList.add('sitting')
    el.append(sittingRoom)

    area.append(el)
    state.squares[squareId] = el
  }
}

function plot(data) {
  let svg = select(document, '.map > object').contentDocument
  let layer = select(svg, '#dotslayer')

  let normaldot = select(svg, '#traveldot-normal')
  let terminaldot = select(svg, '#traveldot-place')
  let dangerdot = select(svg, '#traveldot-danger')

  let drawPoint = (pointId, point) => {
    if (point.city) {
      // let place = data.places[point.place]
      // city marks are already in the SVG
      let star = select(svg, '#'+point.place)
      console.assert(star, point.place)
      star.style.cursor = 'pointer'
      star.addEventListener('click', e => {
        showPrices(point.place)
      })
      return
    }

    let [x, y] = splitDotId(pointId)

    if (point.terminal) {
      let ndot = terminaldot.cloneNode(true)
      ndot.id = "dot-"+pointId
      ndot.setAttributeNS(null, 'x', x-10);
      ndot.setAttributeNS(null, 'y', y-10);
      ndot.style.cursor = 'pointer'
      ndot.addEventListener('click', e => {
        showPrices(point.place)
      })
      layer.append(ndot)
    } else {
      let ndot = null
      if (point.danger) {
        ndot = dangerdot.cloneNode()
      } else {
        ndot = normaldot.cloneNode()
      }
      ndot.id = "dot-"+pointId
      ndot.setAttributeNS(null, 'cx', x);
      ndot.setAttributeNS(null, 'cy', y);
      ndot.addEventListener('click', e => { alert(pointId) })
      layer.append(ndot)
    }
  }

  for (let pointId in data.dots) {
    drawPoint(pointId, data.dots[pointId])
  }
}

function scrollingMap() {
  let pos = { top: 0, left: 0, x: 0, y: 0 }
  let map = select(document, '.map')
  map.addEventListener('mousedown', e => {
    pos.top = map.scrollTop
    pos.left = map.scrollLeft
    pos.x = e.clientX
    pos.y = e.clientY

    let end = new AbortController()

    document.addEventListener('mousemove', e => {
      e.stopPropagation()
      let dx = e.clientX - pos.x
      let dy = e.clientY - pos.y
      map.scrollLeft = pos.left - dx
      map.scrollTop = pos.top - dy
    }, { signal: end.signal, capture: true  })
    document.addEventListener('mouseup', e => {
      e.stopPropagation()
      end.abort()
    }, { once: true, capture: true })
  })
}

// showing messages

function log(msg) {
  let s = select(document, '.messages')
  let d = document.createElement('div')
  if (msg.who) {
    let player = state.players.get(msg.who)
    let where = ""
    if (msg.where) {
      let dotId = msg.where
      let dot = state.data.dots[dotId]
      let place = state.data.places[dot.place]
      if (place) {
        where = "in " + place.name
      } else {
        where = "at " + dotId
      }
    }
    d.innerHTML = `<span style="color: ${player.colour}; font-weight: bold;">${player.name}</span> ${msg.what} ${where}`
    if (msg.who != state.name) {
      showLogLine(d.cloneNode(true))
    }
  } else if (msg.what) {
    d.textContent = msg.what
  } else {
    let text = typeof msg === 'string' ? msg : JSON.stringify(msg)
    d.textContent = text
  }
  s.prepend(d)
}

let messages = null
function showLogLine(line) {
  if (!messages) {
    messages = [line]
    showOneLogLine()
  } else {
    messages.push(line)
  }
}

function showOneLogLine() {
  let ele = select(document, '.showmessage')
  let line = messages.pop()
  if (line) {
    ele.classList.add('open')
    let ine = select(ele, '.message')
    ine.replaceChildren(line)
    setTimeout(showOneLogLine, 2000)
  } else {
    ele.classList.remove('open')
    messages = null
  }
}

// main()

document.addEventListener('DOMContentLoaded', function() {
  let urlParams = new URLSearchParams(window.location.search)
  let gameId = urlParams.get('gameId')
  let name = urlParams.get('name')
  let colour = urlParams.get('colour')

  if (!gameId || !name) {
    alert('missing params')
    return
  }
  if (!colour) {
    // if colour is null, then just observe
    colour = ""
  }

  let mapObject = document.createElement('object')
  mapObject.type = 'image/svg+xml'
  mapObject.data = 'map.svg'
  select(document, '.map').append(mapObject)

  mapObject.addEventListener('load', e => {
    fetch('data.json').
      then(rez => rez.json()).
      then(data => {
        setup(data, gameId, name, colour)
      })
    })

  window.cheat = function(cmd) {
    doRequest('play', { command: 'cheat', options: cmd }, console.log)
  }
})
