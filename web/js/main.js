
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

function connect(name, colour, then) {
  if (state.ws) return

  const conn = new WebSocket(`ws://${location.host}/ws?name=${name}&colour=${colour}`, 'comms')

  conn.onclose = ev => {
    console.log(`WebSocket Disconnected code: ${ev.code}, reason: ${ev.reason}`)
    // if (ev.code !== 1001) {
    //   console.log("Reconnecting in 1s")
    //   setTimeout(dial, 1000)
    // }
  }

  conn.onopen = ev => {
    console.info("websocket connected")
    state.ws = conn
  }

  conn.onmessage = ev => {
    if (typeof ev.data !== "string") {
      console.error("unexpected message type", typeof ev.data)
      return
    }
    console.log(ev.data)
    let msg = JSON.parse(ev.data)
    if (msg.head === 'update') {
      receiveUpdate(msg.data)
    } else if (msg.head === 'turn') {
      receiveTurn(msg.data)
    } else if (msg.head === 'text') {
      log(msg.data)
    } else if (msg.head.startsWith('response:')) {
      let rn = msg.head.substring(9)
      let res = msg.data
      let then = state.reqs.get(rn)
      state.reqs.delete(rn)
      let err = null
      then(msg.data.error, msg.data)
    }
  }
}

function send(type, data) {
  if (!state.ws) return

  let msg = {
    Head: type,
    Data: data
  }

  let jtext = JSON.stringify(msg)
  console.log('sending', msg)
  state.ws.send(jtext)
}

function receiveUpdate(st) {
  if (!st.playing) {
    document.body.setAttribute('started', false)
  } else {
    document.body.setAttribute('started', true)
  }

  if (state.playing != st.playing) {
    // turn has moved on
    state.playing = st.playing
  }

  for (let pl of st.players) {
    let prev = state.players.get(pl.name) || {}

    if (pl.name == state.name) {
      // this is us
      receiveLucks(pl.lucks)
      receiveTicket(pl.ticket)
      receiveStatus(pl.money, pl.souvenirs)
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

  if (!lucks) {
    document.body.setAttribute('hasluck', false)
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
  }
}

function receiveTicket(ticket) {
  if (!ticket) {
    document.body.setAttribute('ticket', false)
  } else {
    let div = select(document, '.ticket')
    select(div, '.by > span').textContent = ticket.by
    let from = state.data.places[ticket.from].name
    select(div, '.from > span').textContent = from
    let to = state.data.places[ticket.to].name
    select(div, '.to > span').textContent = to
    let currency = state.data.currencies[ticket.currency].name
    select(div, '.fare > span').textContent = `${ticket.fare} ${currency}`
    document.body.setAttribute('ticket', true)
  }
}

function receiveStatus(money, souvenirs) {
  let s = select(document, '.aboutme')

  // XXX - this bit never changes
  let sc = select(s, '.colour')
  sc.style.backgroundColor = state.colour
  let sn = select(s, '.name')
  sn.textContent = state.name

  let text = `${JSON.stringify(money)}, ${souvenirs}`

  let st = select(s, '.text')
  st.textContent = text
}

function receiveTurn(st) {
  state.turn = st

  let canLuck = st.can ? st.can.includes('useluck') : false
  document.body.setAttribute('canluck', canLuck)

  let player = state.players.get(st.player)
  document.body.setAttribute('ontrack', !st.onmap)

  let s = select(document, '.currentturn')

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

  let [x, y] = split(dot)

  let marker = select(svg, '#playerring')
  let nmarker = marker.cloneNode()
  nmarker.id = 'player-' + colour
  nmarker.setAttributeNS(null, 'cx', x);
  nmarker.setAttributeNS(null, 'cy', y);
  nmarker.style.stroke = colour
  // TOOD - colour

  state.mapMarks.set(colour, nmarker)
  layer.append(nmarker)
}

function scrollMapTo(dot) {
  let [x, y] = split(dot)

  let scroller = select(document, '.map')
  let scrollee = scroller.firstElementChild
  let sLeft = (x/1000)*scrollee.offsetWidth-scroller.offsetWidth/2+scrollee.offsetLeft
  let sTop = (y/700)*scrollee.offsetHeight-scroller.offsetHeight/2+scrollee.offsetTop
  scroller.scrollTo({ top: sTop, left: sLeft, behavior: 'smooth' })
}

function select(parent, selector) {
  return parent.querySelector(selector)
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

function split(s) {
  let ss = s.split(',')
  return [parseInt(ss[0]), parseInt(ss[1])]
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
      star.addEventListener('click', e => {
        doRequest('query:place:'+point.place, {}, (e, r) => {
          if (e) {
            alert(e.message); return
          }
          log(r)
        })
      })
      return
    }

    let [x, y] = split(pointId)

    if (point.terminal) {
      let ndot = terminaldot.cloneNode(true)
      ndot.id = "dot-"+pointId
      ndot.setAttributeNS(null, 'x', x-10);
      ndot.setAttributeNS(null, 'y', y-10);
      ndot.addEventListener('click', e => {
        doRequest('query:place:'+point.place, {}, (e, r) => {
          if (e) {
            alert(e.message); return
          }
          log(r)
        })
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
}

function makePlayButtons(tgt, actions, clazz) {
  for (let a of actions || []) {
    if (a === 'useluck') {
      // can do this with the cards
      continue
    }

    let cmd = a.split(":")[0]
    let button = document.createElement('button')
    button.classList.add(clazz)
    button.append(cmd)
    button.addEventListener('click', e => {
      doPlay(cmd, state.data.actions[cmd])
    })
    tgt.append(button)
  }
}

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

  let n = 0
  for (let card of stack.querySelectorAll('.luckcard')) {
    card.style.rotate = n + 'turn'
    n += .05
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

function doSelf() {
  doRequest('query:player:'+state.name, {}, (e, r) => {
    if (e) {
      alert(e.message); return
    }
    let lucks = {}
    for (let cardId of r.lucks || []) {
      lucks[cardId] = state.data.lucks[cardId].name
    }
    r.lucks = lucks
    log(r)
  })
}

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

function doPlay(cmd, action) {
  let options = null
  if (action.help) {
    options = prompt(`${cmd} ${action.help}`)
    if (!options) return
  }

  let cb = (e, r) => {
    if (e) { alert(e.message); return; }
  }

  // if (cmd === 'takerisk') {
  //   cb = (e, r) => {
  //     if (e) { alert(e.message); return; }
  //     showRisk(r)
  //   }
  // } else if (cmd === 'takeluck') {
  //   cb = (e, r) => {
  //     if (e) { alert(e.message); return; }
  //     showLuck(r)
  //   }
  // }

  doRequest(`play`, { command: cmd, options: options }, cb)
}

function useLuck(id) {
  let options = prompt('options (or none)')
  options = '' + id + ' ' + options

  let cb = (e, r) => {
    if (e) { alert(e.message); return; }
  }

  doRequest(`play`, { command: 'useluck', options: options }, cb)
}

function showLuck(ind) {
  let div = select(document, '.luck')
  console.log(ind)
}

function showRisk() {
  console.log(ind)
}

function fixup(indata) {
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

function setup(inData, name, colour) {
  state.data = fixup(inData)
  state.name = name
  state.colour = colour

  let startButton = select(document, '#startbutton')
  startButton.addEventListener('click', doStart)

  makeSquares(state.data)
  plot(state.data)
  makeButtons()

  makeLuckStack()

  connect(state.name, state.colour)
}

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
  } else if (msg.what) {
    d.textContent = msg.what
  } else {
    let text = typeof msg === 'string' ? msg : JSON.stringify(msg)
    d.textContent = text
  }
  s.prepend(d)
}

document.addEventListener('DOMContentLoaded', function() {
  let urlParams = new URLSearchParams(window.location.search)
  let name = urlParams.get('name')
  let colour = urlParams.get('colour')

  if (!name || !colour) {
    alert('missing params')
    return
  }

  select(document, 'object').addEventListener('load', e => {
    fetch('data.json').
      then(rez => rez.json()).
      then(data => {
        setup(data, name, colour)
      })
    })
})
