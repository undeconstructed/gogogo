
let state = {
  data: null,
  ws: null,

  reqNo: 0,
  reqs: new Map(),

  squares: [],
  trackMarks: new Map(),
  mapMarks: new Map()
}

function connect(name, colour, then) {
  if (state.ws) return

  const conn = new WebSocket(`ws://${location.hostname}:1235/?name=${name}&colour=${colour}`, 'comms')

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
    if (msg.head === 'turn') {
      receiveState(msg.data)
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

function receiveState(st) {
  let s = select(document, '.state')
  let sc = select(s, '.colour')
  sc.style.backgroundColor = st.colour
  let sn = select(s, '.name')
  sn.textContent = st.player
  let sr = select(s, '.text')
  sr.textContent = JSON.stringify(st)

  markOnTrack(st.colour, st.square)
  markOnMap(st.colour, st.dot)
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
  squareDiv.append(mark)

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
  delete nmarker.id
  nmarker.setAttributeNS(null, 'cx', x);
  nmarker.setAttributeNS(null, 'cy', y);
  nmarker.style.stroke = colour
  // TOOD - colour

  state.mapMarks.set(colour, nmarker)
  layer.append(nmarker)

  // marker.scrollIntoView({ behavior: 'smooth' })
}

function select(parent, selector) {
  return parent.querySelector(selector)
}

function makeSquares(data) {
  let area = select(document, '.squares')
  for (let squareId in data.squares) {
    let square = data.squares[squareId]
    let el = document.createElement('div')
    el.append(square.name)
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
  let dangerdot = select(svg, '#traveldot-danger')

  let drawPoint = (pointId, point) => {
    let [x, y] = split(pointId)
    let ndot = null
    if (point.danger) {
      ndot = dangerdot.cloneNode()
    } else {
      ndot = normaldot.cloneNode()
    }
    delete ndot.id
    ndot.setAttributeNS(null, 'cx', x);
    ndot.setAttributeNS(null, 'cy', y);
    ndot.title = pointId
    ndot.addEventListener('click', e => { alert(pointId) })
    layer.append(ndot)
  }

  for (let pointId in data.dots) {
    drawPoint(pointId, data.dots[pointId])
  }
}

function makeButtons(data) {
  let buttonBox = select(document, '.actions')

  // start button
  {
    let button = document.createElement('button')
    button.append('start')
    button.addEventListener('click', doStart)
    buttonBox.append(button)
  }

  // play action buttons
  for (let a of Object.keys(data.actions)) {
    let button = document.createElement('button')
    button.append(a)
    button.addEventListener('click', e => {
      doPlay(a, data.actions[a])
    })
    buttonBox.append(button)
  }
}

function doRequest(rtype, body, then) {
  let rn = '' + state.reqNo++
  let mtype = "request:" + rn + ":" + rtype
  state.reqs.set(rn, then)
  send(mtype, body)
}

function doStart() {
  return doRequest("start", null, (e, r) => {
    if (e) {
      alert(e.message); return
    }
    // log(JSON.stringify(r))
  })
}

function doPlay(cmd, action) {
  let options = null
  if (action.help) {
    options = prompt(`${cmd} ${action.help}`)
  }
  doRequest(`play`, { command: cmd, options: options }, (e, r) => {
    if (e) {
      alert(e.message); return
    }
    // log(JSON.stringify(r))
  })
}

function setup(inData) {
  state.data = inData

  makeSquares(state.data)
  // img.contentDocument.addEventListener('load', e => {
    plot(state.data)
  // })
  makeButtons(state.data)

  connect('web', 'green')
}

function log(text) {
  let s = select(document, '.messages')
  let d = document.createElement('div')
  d.textContent = text
  s.prepend(d)
}

document.addEventListener('DOMContentLoaded', function() {
  fetch('../data.json').
    then(rez => rez.json()).
    then(data => {
      setup(data)
    })
})
