
// utils

function select(parent, selector) {
  let e = parent.querySelector(selector)
  if (e == null) {
    console.log('null', parent, selector)
    return
  }
  return e
}

// net stuff

let netState = {
  ws: null,

  reqNo: 0,
  reqs: new Map(),

  send() {},
  doRequest() {},
}

function connect(listener, args) {
  if (netState.ws) return

  const conn = new WebSocket(`ws://${location.host}/ws?game=${args.gameId}&name=${args.name}&colour=${args.colour}`, 'comms')

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
      let u = processUpdate(msg.data)
      listener.onUpdate(u)
    } else if (msg.head === 'turn') {
      let t = processTurn(msg.data)
      listener.onTurn(t)
    } else if (msg.head === 'text') {
      listener.onText(msg.data)
    } else if (msg.head.startsWith('response:')) {
      let rn = msg.head.substring(9)
      let then = netState.reqs.get(rn)
      netState.reqs.delete(rn)

      let res = msg.data
      // XXX - nothing says these fields must exist
      then(res.error, res)
    }
  }
}

// game utils

function makeByLine(data, modes) {
  let ms = []
  for (let m of modes) {
    ms.push(data.modes[m])
  }
  return ms.join('/')
}

function splitDotId(s) {
  let ss = s.split(',')
  return [parseInt(ss[0]), parseInt(ss[1])]
}

// receiving data

function promoteCustom(o) {
  for (let x in o.custom) {
    o[x] = o.custom[x]
  }
  delete o.custom
  return o
}

function processUpdate(u) {
  promoteCustom(u)

  u.players = u.players || {}

  for (let pl of u.players) {
    promoteCustom(pl)

    pl.souvenirs = pl.souvenirs || []
    pl.lucks = pl.lucks || []
    pl.money = pl.money || {}

    for (let k in pl.money) {
      if (pl.money[k] == 0) {
        delete pl.money[k]
      }
    }
  }

  return u
}

function processTurn(t) {
  promoteCustom(t)

  t.can = t.can  || []
  t.must = t.must || []

  return t
}

// actions

function doSay() {
  let msg = prompt('Say what?')
  if (!msg) return
  netState.send('text', msg)
}

function doStart() {
  let cb = (e, _r) => { if (e) { alert(e.message); return; } }

  netState.doRequest('start', null, cb)
}

function doPlay(cmd, action, cb) {
  let options = null
  if (action && action.help) {
    options = prompt(`${cmd} ${action.help}`)
    if (options === null) {
      setTimeout(() => {
        cb({ e: { message: 'cancelled' } })
      }, 0)
      return
    }
  }

  if (options) {
    cmd += ':' + options
  }

  netState.doRequest('play', { command: cmd }, cb)
}

function useLuck(id) {
  let options = prompt('options (or none)')

  let cb = (e, _r) => { if (e) { alert(e.message); return; } }

  netState.doRequest('play', { command: 'useluck:'+id+':'+options }, cb)
}

function doChangeMoney(from, to, n) {
  let cb = (e, _r) => { if (e) { alert(e.message); return; } }

  netState.doRequest('play', { command: `changemoney:${from}:${to}:${n}` }, cb)
}

function doDeclareSouvenir(placeId) {
  let cb = (e, _r) => { if (e) { alert(e.message); return; } }

  netState.doRequest('play', { command: 'declare:'+placeId }, cb)
}

function doDiceMove() {
  let cb = (e, r) => {
    if (e) { alert(e.message); return; }
    showLogLine('you rolled a ' + r.message)
  }

  netState.doRequest('play', { command: 'dicemove' }, cb)
}

function doStop() {
  let cb = (e, _r) => { if (e) { alert(e.message); return; } }

  netState.doRequest('play', { command: 'stop' }, cb)
}

function doEnd() {
  let cb = (e, _r) => { if (e) { alert(e.message); return; } }

  netState.doRequest('play', { command: 'end' }, cb)
}

function doGamble(currency, amount) {
  let cb = (e, r) => {
    if (e) { alert(e.message); return; }
    showLogLine('you gambled a ' + r.message)
  }

  netState.doRequest('play', { command: `gamble:${currency}:${amount}` }, cb)
}

function doPay(currency, amount) {
  let cb = (e, _r) => { if (e) { alert(e.message); return; } }

  netState.doRequest('play', { command: `pay:${currency}:${amount}` }, cb)
}

// ui components

function makeStartButton() {
  let shield = document.querySelector('.nostate')
  let startButton = shield.querySelector('#startbutton')

  startButton.addEventListener('click', doStart)

  let onUpdate = s => {
    // TODO - the other statesa aren't handled
    let started = s.status !== 'unstarted'
    shield.classList.toggle('hide', started)
  }

  return { onUpdate }
}

function makeStatusBar() {
  let aboutMe = select(document, '.aboutme')
  let aboutGame = select(document, '.aboutgame')

  let doUpdatePlayers = (players, playing) => {
    let playersDiv = select(aboutGame, '.players > div')
    playersDiv.replaceChildren()
    for (let name in players) {
      let pl = players[name]

      if (pl.name == playing) {
        let sc = select(aboutGame, '.aplayer .colour')
        sc.style.backgroundColor = pl.colour
        let sn = select(aboutGame, '.aplayer .name')
        sn.textContent = pl.name
      }

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
      playersDiv.append(div)
    }
  }

  let onUpdate = s => {
    doUpdatePlayers(s.players, s.playing)

    // XXX - this can't change
    let sc = select(aboutMe, '.colour')
    sc.style.backgroundColor = s.me.colour
    let sn = select(aboutMe, '.name')
    sn.textContent = s.me.name
  }

  return { onUpdate }
}

function makeMap(data, up) {
  let marks = {}

  let svg = select(document, '.map > object').contentDocument
  let layer = select(svg, '#dotslayer')

  ;(() => {
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
        star.addEventListener('click', _e => {
          up.send({ do: 'showprices', at: point.place })
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
        ndot.addEventListener('click', _e => {
          up.send({ do: 'showprices', at: point.place })
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
        ndot.addEventListener('click', _e => { alert(pointId) })
        layer.append(ndot)
      }
    }

    for (let pointId in data.dots) {
      drawPoint(pointId, data.dots[pointId])
    }
  })()

  let makeMark = (colour, dot) => {
    if (!colour) {
      return
    }

    let prev = marks[colour]
    if (prev) {
      prev.remove()
    }

    let [x, y] = splitDotId(dot)

    let marker = select(svg, '#playerring')
    let nmarker = marker.cloneNode()
    nmarker.id = 'player-' + colour
    nmarker.setAttributeNS(null, 'cx', x);
    nmarker.setAttributeNS(null, 'cy', y);
    nmarker.style.stroke = colour

    marks[colour] = nmarker
    layer.append(nmarker)
  }

  let makeMarks = (players, playing) => {
    for (let name in players) {
      let pl = players[name]
      makeMark(pl.colour, pl.dot)
      if (pl.name == playing) {
        scrollTo(pl.dot)
      }
    }
  }

  let scrollTo = (dot) => {
    let [x, y] = splitDotId(dot)

    let scroller = select(document, '.map')
    let scrollee = scroller.firstElementChild
    let sLeft = (x/1000)*scrollee.offsetWidth-scroller.offsetWidth/2+scrollee.offsetLeft
    let sTop = (y/700)*scrollee.offsetHeight-scroller.offsetHeight/2+scrollee.offsetTop
    scroller.scrollTo({ top: sTop, left: sLeft, behavior: 'smooth' })
  }

  // scrollingMap()

  let onUpdate = s => {
    makeMarks(s.players, s.playing)
  }

  return { onUpdate }
}

function makeSquares(data) {
  let div = select(document, '.squares')
  let squares = []
  let marks = {}

  ;(() => {
    let z = 1000

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

      div.append(el)
      squares[squareId] = el
    }
  })()

  let makeMark = (colour, square) => {
    if (!colour) {
      return
    }

    let prev = marks[colour]
    if (prev) {
      prev.remove()
    }

    let mark = document.createElement('div')
    mark.classList.add('mark')
    mark.style.backgroundColor = colour
    let squareDiv = squares[square]

    marks[colour] = mark
    select(squareDiv, '.sitting').append(mark)
  }

  let makeMarks = (players, playing) => {
    for (let name in players) {
      let pl = players[name]
      makeMark(pl.colour, pl.square)
      if (pl.name == playing) {
        scrollTo(pl.square)
      }
    }
  }

  let scrollTo = (square) => {
    let squareDiv = squares[square]
    squareDiv.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }

  let onUpdate = s => {
    makeMarks(s.players, s.playing)
  }

  let onTurn = t => {
    div.classList.toggle('focus', !t.onmap)
  }

  return { onUpdate, onTurn }
}

function makePriceList(data) {
  let ele = select(document, '.prices')

  ;(() => {
    let tbody = select(ele, 'tbody')
    tbody.replaceChildren()

    let makePlacePrices = (placeId) => {
      let place = data.places[placeId]
      let currency = data.currencies[place.currency]

      let stRate = data.currencies['st'].rate
      let loRate = currency.rate

      let linen = 0
      for (let r in place.routes) {
        let tr = document.createElement('tr')
        let th = document.createElement('th')
        if (linen == 0) {
          th.classList.add('place')
          th.classList.add(placeId)
          th.textContent = place.name
        } else if (linen == 1) {
          th.textContent = `(${currency.name})`
        }
        tr.append(th)
        linen++

        let ss = r.split(':')
        let destId = ss[0]
        let dest = data.places[destId].name
        let modeId = ss[1]
        let mode = makeByLine(data, modeId)
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

        tr.addEventListener('click', _e => {
          let cb = (e, _r) => {
            if (e) { alert(e.message); return; }
            showLogLine('you have bought a ticket')
          }

          doPlay(`buyticket:${placeId}:${destId}:${modeId}`, null, cb)
        })

        tbody.append(tr)
      }

      let sep = document.createElement('tr')
      sep.classList.add('sep')
      tbody.append(sep)
    }

    for (let placeId in data.places) {
      makePlacePrices(placeId)
    }
  })()

  let doOpen = (placeId) => {
    ele.classList.add('open')
    let p = select(ele, `.${placeId}`)
    p.scrollIntoView({ behavior: 'smooth' })
  }

  let doClose = () => {
    ele.classList.remove('open')
  }

  ele.addEventListener('click', doClose)

  let onCommand = c => {
    if (c.do === 'showprices') {
      doOpen(c.at)
    }
  }

  let onTurn = t => {
    let canbuy = false
    for (let cmd of t.can) {
      if (cmd.startsWith('buyticket:')) {
        canbuy = true
      }
    }
    ele.classList.toggle('canbuy', canbuy)
  }

  return { onCommand, onTurn }
}

function makeLuckView(data) {
  let div = select(document, '.showluck')

  let doTakeSetup = () => {
    div.classList.add('blank')
    div.classList.add('back')
    div.classList.add('open')

    let button = select(div, '.button')
    button.addEventListener('click', _e => {
      let cb = (e, r) => {
        if (e) {
          alert(e.message)
          doTakeSetup()
          return
        }
        doShow(r.message)
      }
      netState.doRequest('play', { command: 'takeluck' }, cb)
    }, { once: true })
  }

  let doShow = cardId => {
    let card = data.lucks[cardId]
    if (!card) {
      card = {
        name: 'there are only so many cards'
      }
    }

    div.classList.remove('back')
    div.classList.remove('blank')
    select(div, '.card .body').textContent = card.name

    div.addEventListener('click', _e => {
      doClose()
    }, { once: true })
  }

  let doClose = () => {
    div.classList.remove('open')
  }

  let onTurn = t => {
    if (t.hasMust('takeluck')) {
      doTakeSetup()
    }
  }

  return { onTurn }
}

function makeRiskView(data) {
  let div = select(document, '.showrisk')

  let doTakeSetup = () => {
    div.classList.add('blank')
    div.classList.add('back')
    div.classList.add('open')

    let button = select(div, '.button')
    button.addEventListener('click', _e => {
      let cb = (e, r) => {
        if (e) {
          alert(e.message)
          doTakeSetup()
          return
        }
        doShow(r.message)
      }
      netState.doRequest('play', { command: 'takerisk' }, cb)
    }, { once: true })
  }

  let doShow = cardId => {
    let card = data.risks[cardId]

    let div = select(document, '.showrisk')
    let foot = select(div, '.foot')
    foot.replaceChildren()

    div.classList.remove('back')
    div.classList.remove('blank')
    select(div, '.card .body').textContent = card.name

    // not closeable until turn arrives to say whether we must obey
  }

  let doShowForObey = (cardId, canIgnore) => {
    // XXX - this will probably interrupt the turn animation
    div.classList.add('open')
    div.classList.remove('back')
    div.classList.remove('blank')

    let card = data.risks[cardId]
    select(div, '.card .body').textContent = card.name

    let foot = select(div, '.foot')
    foot.replaceChildren()

    let buttons = []

    if (canIgnore) {
      let ignoreButton = document.createElement('button')
      ignoreButton.append('[ignore]')
      ignoreButton.addEventListener('click', _e => {
        doClose()
        let cb = (e, _r) => {
          if (e) {
            doShowForObey(cardId)
            alert(e.message)
            return
          }
        }
        netState.doRequest('play', { command: 'ignorerisk:'+cardId }, cb)
      }, { once: true })
      buttons.push(ignoreButton)
    }

    let obeyButton = document.createElement('button')
    obeyButton.append('[obey]')
    obeyButton.addEventListener('click', _e => {
      doClose()
      let cb = (e, _r) => {
        if (e) {
          doShowForObey(cardId)
          alert(e.message)
          return
        }
      }
      netState.doRequest('play', { command: 'obeyrisk:'+cardId }, cb)
    }, { once: true })
    buttons.push(obeyButton)

    foot.replaceChildren(...buttons)
  }

  let doClose = () => {
    div.classList.remove('open')
  }

  let onTurn = t => {
    let take = false, obey = -1, ignore = -1
    for (let cmd of t.can) {
      if (cmd.startsWith('ignorerisk:')) {
        ignore = parseInt(cmd.split(':')[1])
      }
    }
    for (let cmd of t.must) {
      if (cmd === 'takerisk') {
        take = true
      } else if (cmd.startsWith('obeyrisk:')) {
        obey = parseInt(cmd.split(':')[1])
      }
    }
    if (take) {
      doTakeSetup()
    } else if (obey >= 0) {
      doShowForObey(obey, ignore == obey)
    } else if (div.classList.contains('open')) {
      // is open, but don't have to do anything
      div.addEventListener('click', _e => {
        doClose()
      }, { once: true })
    }
  }

  return { onTurn }
}

function makeLuckStack(data) {
  let stack = select(document, '.lucks')

  let doOpen = () => {
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

  let doClose = () => {
    stack.classList.remove('open')
    stack.classList.add('stashed')

    for (let card of stack.querySelectorAll('.luckcard')) {
      card.style.rotate = 'unset'
    }
  }

  let doReceive = (lucks) => {
    stack.replaceChildren()

    if (lucks.length == 0) {
      stack.classList.add('empty')
      doClose()
    } else {
      stack.classList.remove('empty')

      let tmpl = select(document, '#lucktemplate').content.firstElementChild

      for (let luckId of lucks) {
        let luckData = data.lucks[luckId]
        let div = tmpl.cloneNode(true)
        select(div, '.body').textContent = luckData.name
        select(div, 'button').addEventListener('click', e => {
          e.stopPropagation()
          useLuck(luckId)
          doClose()
        })
        stack.append(div)
      }

      if (stack.classList.contains('open')) {
        doOpen()
      }
    }
  }

  stack.addEventListener('click', _e => {
    if (stack.classList.contains('stashed')) {
      doOpen()
    } else {
      doClose()
    }
  })

  let onUpdate = s => {
    doReceive(s.me.lucks)
  }

  let onTurn = t => {
    stack.classList.toggle('canluck', t.hasCan('useluck:*'))
  }

  return { onUpdate, onTurn }
}

function makeMoneyPile(data, up) {
  let stack = select(document, '.money')
  let tmpl = select(document, '#moneytemplate').content.firstElementChild

  let changeTo = null

  let doOpen = () => {
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

  let doClose = () => {
    stack.classList.remove('open')
    stack.classList.add('stashed')

    for (let card of stack.querySelectorAll('.banknote')) {
      card.style.rotate = 'unset'
    }
  }

  let doReceive = (money) => {
    stack.replaceChildren()

    if (Object.keys(money).length == 0) {
      stack.classList.add('empty')
      doClose()
    } else {
      stack.classList.remove('empty')
      for (let cId in money) {
        let amount = money[cId]
        if (amount) {
          let currency = data.currencies[cId]
          let div = tmpl.cloneNode(true)
          div.classList.add(cId)
          select(div, '.head').textContent = cId // currency.name
          select(div, '.body').textContent = '' + money[cId]
          div.style.backgroundColor = currency.colour
          // if (cId === 'ye') {
          //   select(div, '.inner').style.backgroundImage = 'url(img/money_ye.svg)'
          // }
          select(div, 'button.change').addEventListener('click', e => {
            e.stopPropagation()
            let ns = prompt('how much?')
            if (!ns) { return; }
            let n = parseInt(ns)
            doChangeMoney(cId, changeTo, n) // TODO - callback
            doClose()
          })
          if (cId !== 'tc') {
            // traveller's cheques can only be changed
            select(div, 'button.pay').addEventListener('click', e => {
              e.stopPropagation()
              let ns = prompt('how much?')
              if (!ns) { return; }
              let n = parseInt(ns)
              doPay(cId, n) // TODO - callback
              doClose()
            })
            select(div, 'button.gamble').addEventListener('click', e => {
              e.stopPropagation()
              let ns = prompt('how much?')
              if (!ns) { return; }
              let n = parseInt(ns)
              // doGamble(cId, n) // TODO - callback
              up.send({ do: 'gamble', currency: cId, amount: n })
              doClose()
            })
          }
          stack.append(div)
        }
      }

      if (stack.classList.contains('open')) {
        doOpen()
      }
    }
  }

  stack.addEventListener('click', _e => {
    if (stack.classList.contains('stashed')) {
      doOpen()
    } else {
      doClose()
    }
  })

  let onUpdate = s => {
    doReceive(s.me.money)
  }

  let onTurn = t => {
    changeTo = null
    let gamble = false, pay = false
    for (let cmd of t.can) {
      if (cmd.startsWith('changemoney:')) {
        let to = cmd.substring(14, 16) // XXX - horrible
        changeTo = to
      } else if (cmd.startsWith('gamble:')) {
        gamble = true
      }
    }
    for (let cmd of t.must) {
      if (cmd.startsWith('pay:')) {
        pay = true
      }
    }
    stack.classList.toggle('canchange', changeTo != null)
    stack.classList.toggle('cangamble', gamble)
    stack.classList.toggle('canpay', pay)
  }

  return { onUpdate, onTurn }
}

function makeTicketView(data) {
  let div = document.querySelector('.ticket')

  let doReceive = (ticket) => {
    if (!ticket) {
      div.classList.add('empty')
    } else {
      div.classList.remove('empty')
      select(div, '.by > span').textContent = makeByLine(data, ticket.by)
      let from = data.places[ticket.from].name
      select(div, '.from > span').textContent = from
      let to = data.places[ticket.to].name
      select(div, '.to > span').textContent = to
      let currency = data.currencies[ticket.currency].name
      select(div, '.fare > span').textContent = `${ticket.fare} ${currency}`
    }
  }

  let onUpdate = s => {
    doReceive(s.me.ticket)
  }

  return { onUpdate }
}

function makeSouvenirPile(data) {
  let stack = select(document, '.souvenirs')
  let tmpl = select(document, '#souvenirtemplate').content.firstElementChild

  let doOpen = () => {
    stack.classList.remove('stashed')
    stack.classList.add('open')

    let move = -3

    let n = 0
    for (let card of stack.querySelectorAll('.souvenircard')) {
      card.style.left = n + 'rem'
      n += move
    }
  }

  let doClose = () => {
    stack.classList.remove('open')
    stack.classList.add('stashed')

    for (let card of stack.querySelectorAll('.souvenircard')) {
      card.style.left = 0
    }
  }

  let doReceive = (souvenirs) => {
    stack.replaceChildren()

    if (souvenirs.length == 0) {
      stack.classList.add('empty')
      doClose()
    } else {
      stack.classList.remove('empty')

      for (let placeId of souvenirs) {
        let place = data.places[placeId]
        let currency = data.currencies[place.currency]
        let div = tmpl.cloneNode(true)
        select(div, '.where').textContent = 'Souvenir from ' + place.name
        let price = data.settings.souvenirPrice * currency.rate
        select(div, '.price').textContent = '' + price + ' ' + currency.name
        for (let bar of div.querySelectorAll('.bar')) {
          bar.style.backgroundColor = currency.colour
        }
        select(div, 'button').addEventListener('click', e => {
          e.stopPropagation()
          doDeclareSouvenir(placeId)
          doClose()
        })
        stack.append(div)
      }

      if (stack.classList.contains('open')) {
        doOpen()
      }
    }
  }

  stack.addEventListener('click', _e => {
    if (stack.classList.contains('stashed')) {
      doOpen()
    } else {
      doClose()
    }
  })

  let onUpdate = s => {
    if (s.me.souvenirs) {
      doReceive(s.me.souvenirs)
    }
  }

  let onTurn = t => {
    stack.classList.toggle('mustdeclare', t.hasMust('declare:*'))
  }

  return { onUpdate, onTurn }
}

function makeAutoButtons(data) {
  let buttonBox = select(document, '.actions')

  let doOpen = () => {
    buttonBox.classList.add('open')
  }

  let doClose = () => {
    buttonBox.classList.remove('open')
  }

  let doReceive = (turn) => {
    buttonBox.replaceChildren()

    if (turn) {
      makePlayButtons(buttonBox, turn.can, 'can')
      makePlayButtons(buttonBox, turn.must, 'must')
      doOpen()
    }
  }

  let makePlayButtons = (tgt, actions, clazz) => {
    for (let a of actions || []) {
      let parts = a.split(":")

      let cmd = parts[0]
      let action = data.actions[cmd]

      let button = document.createElement('button')
      button.classList.add(clazz)

      let cb = null

      if (cmd === 'useluck') {
        // can do this with the cards
        continue
      } else if (cmd === 'buyticket') {
        // use the price list
        continue
      } else if (cmd === 'changemoney' || cmd === 'pay') {
        // can do this with the notes
        continue
      } else if (cmd === 'declare') {
        // continue
        // can declare with the cards, but declare:none is harder
        button.classList.add('text')
        button.append(cmd)
      } else if (cmd === 'dicemove' || cmd === 'stop' || cmd === 'end') {
        // there's a big button somewhere
        continue
      } else if (cmd === 'gamble') {
        // money + big button
        continue
      } else if (cmd === 'takeluck') {
        // luckview component does this automatically
        continue
      } else if (cmd === 'takerisk' || cmd === 'obeyrisk' || cmd === 'ignorerisk') {
        // riskview component does this automatically
        continue
      } else if (cmd === 'buysouvenir') {
        button.classList.add('buysouvenir')
        // we know this command is complete, so no prompt
        cmd = a
        action = null
        cb = r => showLogLine('you have bought a ' + r.message)
      } else {
        button.classList.add('text')
        button.append(cmd)
      }

      let cb1 = (e, r) => {
        if (e) {
          alert(e.message)
          doOpen()
          return
        }
        cb(r)
      }

      button.addEventListener('click', _e => {
        doClose()
        doPlay(cmd, action, cb1)
      })
      tgt.append(button)
    }
  }

  let onTurn = t => {
    doReceive(t)
  }

  return { onTurn }
}

function makeStopButton() {
  let div = select(document, '.stopbutton')

  ;(() => {
    div.addEventListener('click', _e => {
      doStop()
      div.classList.remove('open')
    })
  })()

  let onTurn = t => {
    div.classList.toggle('open', t.hasCan('stop'))
  }

  return { onTurn }
}

function makeDiceButton() {
  let div = select(document, '.dicebutton')

  ;(() => {
    div.addEventListener('click', _e => {
      doDiceMove()
      div.classList.remove('open')
    })
  })()

  let onTurn = t => {
    div.classList.toggle('open', t.hasCan('dicemove'))
  }

  return { onTurn }
}

function makeGambleButton() {
  let div = document.querySelector('.gamblebutton')

  let currency = null, amount = -1

  ;(() => {
    div.addEventListener('click', _e => {
      doGamble(currency, amount)
      div.classList.remove('open')
    })
  })()

  let onCommand = c => {
    if (c.do === 'gamble') {
      currency = c.currency
      amount = c.amount
      div.classList.add('open')
    }
  }

  let onTurn = _t => {
    div.classList.remove('open')
    currency = null
    amount = -1
  }

  return { onTurn, onCommand }
}

function makeSleepButton() {
  let div = select(document, '.sleepbutton')

  ;(() => {
    div.addEventListener('click', _e => {
      doEnd()
      div.classList.remove('open')
    })
  })()

  let onTurn = t => {
    div.classList.toggle('open', t.hasCan('end'))
  }

  return { onTurn }
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

function newUI(data, gameId, name, colour) {
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
    state.playing = u.playing
    for (let pl of u.players) {
      state.players[pl.name] = pl
      if (pl.name == state.me.name) {
        state.me = pl
      }
    }
    for (let n of u.news) {
      doLog(state, n)
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
  }

  let onTurn = t => {
    turn = newTurn(t)
    for (let c of components) {
      c.onTurn && c.onTurn(turn)
    }
  }

  let onText = t => {
    doLog(state, t)
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

function setup(inData, gameId, name, colour) {
  let data = fixupData(inData)

  let ui = newUI(data, gameId, name, colour)
  window.ui = ui

  ui.addComponent(makeStartButton)
  ui.addComponent(makeStatusBar)
  ui.addComponent(makeMap)
  ui.addComponent(makeSquares)
  ui.addComponent(makePriceList)
  ui.addComponent(makeLuckView)
  ui.addComponent(makeRiskView)
  ui.addComponent(makeLuckStack)
  ui.addComponent(makeMoneyPile)
  ui.addComponent(makeTicketView)
  ui.addComponent(makeSouvenirPile)
  ui.addComponent(makeAutoButtons)
  ui.addComponent(makeDiceButton)
  ui.addComponent(makeStopButton)
  ui.addComponent(makeSleepButton)
  ui.addComponent(makeGambleButton)

  // select(document, '.showluck').addEventListener('click', hideLuck)
  // select(document, '.showrisk').addEventListener('click', hideRisk)

  connect(ui, {gameId, name, colour})
}

function scrollingMap() {
  let map = select(document, '.map')
  setupScrolling(map, map)
  // let svg = select(map, 'object').contentDocument.rootElement
  // setupScrolling(svg, map)
}

function setupScrolling(elem, tgt) {
  let pos = { top: 0, left: 0, x: 0, y: 0 }
  let ondown = e => {console.log(e)
    pos.top = tgt.scrollTop
    pos.left = tgt.scrollLeft
    pos.x = e.clientX
    pos.y = e.clientY

    let end = new AbortController()

    // in HTML only, this is enough, but ..
    document.addEventListener('mousemove', e => {
      e.stopPropagation()
      let dx = e.clientX - pos.x
      let dy = e.clientY - pos.y
      tgt.scrollLeft = pos.left - dx
      tgt.scrollTop = pos.top - dy
    }, { signal: end.signal, capture: true  })
    document.addEventListener('mouseup', e => {
      e.stopPropagation()
      end.abort()
    }, { signal: end.signal, once: true, capture: true })

    // the document doesn't receive events from the svg ..
    // but also, the coords are wrong?
    // elem.addEventListener('mousemove', e => {
    //   e.stopPropagation()
    //   let dx = e.clientX - pos.x
    //   let dy = e.clientY - pos.y
    //   tgt.scrollLeft = pos.left - dx
    //   tgt.scrollTop = pos.top - dy
    // }, { signal: end.signal, capture: true  })
    // elem.addEventListener('mouseup', e => {
    //   e.stopPropagation()
    //   end.abort()
    // }, { signal: end.signal, once: true, capture: true })
  }

  elem.addEventListener('mousedown', ondown)
}

// showing messages

function doLog(state, msg) {
  let s = select(document, '.messages')
  let d = document.createElement('div')
  if (msg.who) {
    let player = state.players[msg.who]
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
    if (msg.who != state.me.name) {
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

function main() {
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

  mapObject.addEventListener('load', _e => {
    fetch('data.json').
      then(rez => rez.json()).
      then(data => {
        setup(data, gameId, name, colour)
      })
    })

  window.cheat = function(cmd) {
    netState.doRequest('play', { command: 'cheat', options: cmd }, console.log)
  }
}

document.addEventListener('DOMContentLoaded', main)
