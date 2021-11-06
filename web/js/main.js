
import { newUI, connect, promoteCustom } from './game.js'

// net stuff

let netState = null

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

// game utils

function makeByLine(data, modes) {
  let ms = []
  for (let m of modes) {
    ms.push(data.modes[m])
  }
  return ms.join('/')
}

// ui components

function makeStartButton() {
  let shield = document.querySelector('.nostate')
  let startButton = shield.querySelector('#startbutton')

  startButton.addEventListener('click', _e => {
    let cb = (e, _r) => { if (e) { alert(e.message); return; } }
    netState.doRequest('start', null, cb)
  })

  let onUpdate = s => {
    // TODO - the other statesa aren't handled
    let started = s.status !== 'unstarted'
    shield.classList.toggle('hide', started)
  }

  return { onUpdate }
}

function makeStatusBar() {
  let aboutMe = document.querySelector('.aboutme')
  let aboutGame = document.querySelector('.aboutgame')

  let doUpdatePlayers = (players, playing) => {
    let playersDiv = aboutGame.querySelector('.players > div')
    playersDiv.replaceChildren()
    for (let name in players) {
      let pl = players[name]

      if (pl.name == playing) {
        let sc = aboutGame.querySelector('.aplayer .colour')
        sc.style.backgroundColor = pl.colour
        let sn = aboutGame.querySelector('.aplayer .name')
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
    let sc = aboutMe.querySelector('.colour')
    sc.style.backgroundColor = s.me.colour
    let sn = aboutMe.querySelector('.name')
    sn.textContent = s.me.name
  }

  return { onUpdate }
}

function makeMap(data, up) {
  let marks = {}

  let svg = document.querySelector('.map > object').contentDocument
  let layer = svg.querySelector('#dotslayer')

  let splitDotId = s => {
    let ss = s.split(',')
    return [parseInt(ss[0]), parseInt(ss[1])]
  }

  ;(() => {
    let normaldot = svg.querySelector('#traveldot-normal')
    let terminaldot = svg.querySelector('#traveldot-place')
    let dangerdot = svg.querySelector('#traveldot-danger')

    let drawPoint = (pointId, point) => {
      if (point.city) {
        // let place = data.places[point.place]
        // city marks are already in the SVG
        let star = svg.querySelector('#'+point.place)
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

    let marker = svg.querySelector('#playerring')
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

    let scroller = document.querySelector('.map')
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
  let div = document.querySelector('.squares')
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
    squareDiv.querySelector('.sitting').append(mark)
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

function makePriceList(data, up) {
  let ele = document.querySelector('.prices')

  ;(() => {
    let tbody = ele.querySelector('tbody')
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
            up.send({ do: 'notify', msg: 'you have bought a ticket' })
          }
          netState.doRequest('play', { command: `buyticket:${placeId}:${destId}:${modeId}` }, cb)
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
    let p = ele.querySelector(`.${placeId}`)
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
  let div = document.querySelector('.showluck')

  let doTakeSetup = () => {
    div.classList.add('blank')
    div.classList.add('back')
    div.classList.add('open')

    let button = div.querySelector('.button')
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
    div.querySelector('.card .body').textContent = card.name

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
  let div = document.querySelector('.showrisk')

  let doTakeSetup = () => {
    div.classList.add('blank')
    div.classList.add('back')
    div.classList.add('open')

    let button = div.querySelector('.button')
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

    let div = document.querySelector('.showrisk')
    let foot = div.querySelector('.foot')
    foot.replaceChildren()

    div.classList.remove('back')
    div.classList.remove('blank')
    div.querySelector('.card .body').textContent = card.name

    // not closeable until turn arrives to say whether we must obey
  }

  let doShowForObey = (cardId, canIgnore) => {
    // XXX - this will probably interrupt the turn animation
    div.classList.add('open')
    div.classList.remove('back')
    div.classList.remove('blank')

    let card = data.risks[cardId]
    div.querySelector('.card .body').textContent = card.name

    let foot = div.querySelector('.foot')
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
  let stack = document.querySelector('.lucks')

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

      let tmpl = document.querySelector('#lucktemplate').content.firstElementChild

      for (let luckId of lucks) {
        let luckData = data.lucks[luckId]
        let div = tmpl.cloneNode(true)
        div.querySelector('.body').textContent = luckData.name
        div.querySelector('button').addEventListener('click', e => {
          e.stopPropagation()
          let options = prompt('options (or none)')
          if (options == null) {
            return
          }
          let cb = (e, _r) => { if (e) { alert(e.message); return; } }
          netState.doRequest('play', { command: 'useluck:'+luckId+':'+options }, cb)
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
  let stack = document.querySelector('.money')
  let tmpl = document.querySelector('#moneytemplate').content.firstElementChild

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
          div.querySelector('.head').textContent = cId // currency.name
          div.querySelector('.body').textContent = '' + money[cId]
          div.style.backgroundColor = currency.colour
          // if (cId === 'ye') {
          //   div.querySelector('.inner').style.backgroundImage = 'url(img/money_ye.svg)'
          // }
          div.querySelector('button.change').addEventListener('click', e => {
            e.stopPropagation()
            let ns = prompt('how much?')
            if (!ns) { return; }

            let from = cId
            let to = changeTo
            let amount = parseInt(ns)

            let cb = (e, _r) => { if (e) { alert(e.message); return; } }
            netState.doRequest('play', { command: `changemoney:${from}:${to}:${amount}` }, cb)

            doClose()
          })
          if (cId !== 'tc') {
            // traveller's cheques can only be changed
            div.querySelector('button.pay').addEventListener('click', e => {
              e.stopPropagation()
              let ns = prompt('how much?')
              if (!ns) { return; }

              let currency = cId
              let amount = parseInt(ns)

              let cb = (e, _r) => { if (e) { alert(e.message); return; } }
              netState.doRequest('play', { command: `pay:${currency}:${amount}` }, cb)

              doClose()
            })
            div.querySelector('button.gamble').addEventListener('click', e => {
              e.stopPropagation()
              let ns = prompt('how much?')
              if (!ns) { return; }

              let currency = cId
              let amount = parseInt(ns)

              up.send({ do: 'gamble', currency: currency, amount: amount })

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
      } else if (cmd.startsWith('pay:')) {
        pay = true
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
      div.querySelector('.by > span').textContent = makeByLine(data, ticket.by)
      let from = data.places[ticket.from].name
      div.querySelector('.from > span').textContent = from
      let to = data.places[ticket.to].name
      div.querySelector('.to > span').textContent = to
      let currency = data.currencies[ticket.currency].name
      div.querySelector('.fare > span').textContent = `${ticket.fare} ${currency}`
    }
  }

  let onUpdate = s => {
    doReceive(s.me.ticket)
  }

  return { onUpdate }
}

function makeSouvenirPile(data) {
  let stack = document.querySelector('.souvenirs')
  let tmpl = document.querySelector('#souvenirtemplate').content.firstElementChild

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
        div.querySelector('.where').textContent = 'Souvenir from ' + place.name
        let price = data.settings.souvenirPrice * currency.rate
        div.querySelector('.price').textContent = '' + price + ' ' + currency.name
        for (let bar of div.querySelectorAll('.bar')) {
          bar.style.backgroundColor = currency.colour
        }
        div.querySelector('button').addEventListener('click', e => {
          e.stopPropagation()
          let cb = (e, _r) => { if (e) { alert(e.message); return; } }
          netState.doRequest('play', { command: 'declare:'+placeId }, cb)
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

function makeAutoButtons(data, up) {
  let buttonBox = document.querySelector('.actions')

  let skip = [
    'useluck',
    'buyticket',
    'changemoney',
    // 'declare',
    'pay',
    'dicemove',
    'stop',
    'end',
    'gamble',
    'takeluck',
    'takerisk',
    'obeyrisk',
    'ignorerisk'
  ]
  skip = []

  let doPromptPlay = (cmd, opts, action, cb) => {
    let options = null
    if (action && action.help) {
      options = prompt(`${cmd} ${action.help}`, opts)
      if (options === null) {
        setTimeout(() => {
          cb('cancelled')
        }, 0)
        return
      }
    }

    if (options) {
      cmd += ':' + options
    }

    let [cmd1, options1] = cmd.split(' ')

    netState.doRequest('play', { command: cmd1, options: options1 }, cb)
  }

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

      if (skip.includes(cmd)) {
        continue
      }

      // if (cmd === 'buysouvenir') {
      //   button.classList.add('buysouvenir')
      //   // we know this command is complete, so no prompt
      //   cmd = a
      //   action = null
      //   cb = r => up.send({ do: 'notify', msg: 'you have bought a ' + r.message })
      // } else {
        button.classList.add('text')
        button.append(cmd)
      // }

      let cb1 = (e, r) => {
        if (e) {
          if (e !== 'cancelled') { alert(e.message); }
          doOpen()
          return
        }
        cb && cb(r)
      }

      button.addEventListener('click', _e => {
        doClose()
        let opts = parts.slice(1).join(':')
        doPromptPlay(cmd, opts, action, cb1)
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
  let div = document.querySelector('.stopbutton')

  ;(() => {
    div.addEventListener('click', _e => {
      let cb = (e, _r) => { if (e) { alert(e.message); return; } }
      netState.doRequest('play', { command: 'stop' }, cb)
      div.classList.remove('open')
    })
  })()

  let onTurn = t => {
    div.classList.toggle('open', t.hasCan('stop'))
  }

  return { onTurn }
}

function makeDiceButton(_data, up) {
  let div = document.querySelector('.dicebutton')

  ;(() => {
    div.addEventListener('click', _e => {
      let cb = (e, r) => {
        if (e) { alert(e.message); return; }
        up.send({ do: 'notify', msg: 'you rolled a ' + r.message})
      }
      netState.doRequest('play', { command: 'dicemove' }, cb)
      div.classList.remove('open')
    })
  })()

  let onTurn = t => {
    div.classList.toggle('open', t.hasCan('dicemove'))
  }

  return { onTurn }
}

function makeGambleButton(_data, up) {
  let div = document.querySelector('.gamblebutton')

  let currency = null, amount = -1

  ;(() => {
    div.addEventListener('click', _e => {
      let cb = (e, r) => {
        if (e) { alert(e.message); return; }
        up.send({ do: 'notify', msg: 'you gambled and ' + r.message})
      }
      netState.doRequest('play', { command: `gamble:${currency}:${amount}` }, cb)
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
  let div = document.querySelector('.sleepbutton')

  ;(() => {
    div.addEventListener('click', _e => {
      let cb = (e, _r) => { if (e) { alert(e.message); return; } }
      netState.doRequest('play', { command: 'end' }, cb)
      div.classList.remove('open')
    })
  })()

  let onTurn = t => {
    div.classList.toggle('open', t.hasCan('end'))
  }

  return { onTurn }
}

function makeLog(_data, up) {
  let div = document.querySelector('.messages')
  let state = {}

  let doAddLine = msg => {
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
        up.send({ do: 'notify', msg: d.cloneNode(true) })
      }
    } else if (msg.what) {
      d.textContent = msg.what
    } else {
      let text = typeof msg === 'string' ? msg : JSON.stringify(msg)
      d.textContent = text
    }
    div.prepend(d)
  }

  let onUpdate = s => {
    state = s
  }

  let onCommand = c => {
    if (c.do === 'log') {
      doAddLine(c.msg)
    }
  }

  return { onUpdate, onCommand }
}

function makeNotifier() {
  let div = document.querySelector('.showmessage')
  let closeTimeout = null

  let doShowLine = line => {
    if (closeTimeout) {
      // cancel closeTimeout, to extend the time
      clearTimeout(closeTimeout)
    }

    div.classList.add('open')
    let ine = div.querySelector('.message')
    ine.append(line)

    setTimeout(() => {
      line.remove()
    }, 3000)

    closeTimeout = setTimeout(() => {
      div.classList.remove('open')
    }, 3000)
  }

  let onCommand = c => {
    if (c.do === 'notify') {
      let e = c.msg
      if (!(e instanceof Element)) {
        let d = document.createElement('div')
        d.textContent = e
        e = d
      }
      doShowLine(e)
    }
  }

  return { onCommand }
}

function makeDebt() {
  let div = document.querySelector('.debt')

  let onUpdate = s => {
    let hasDebt = s.me.debt != null
    if (hasDebt) {
      div.textContent = "DEBT: " + s.me.debt.amount
    }
    div.classList.toggle('show', hasDebt)
  }

  return { onUpdate }
}

function makeWindicator() {
  let ele = document.querySelector('.win')

  let doReceive = (status, winner) => {
    let won = status === 'won'
    ele.querySelector('.inner').textContent = `${winner} wins!`
    ele.classList.toggle('show', won)
  }

  let onUpdate = s => {
    doReceive(s.status, s.winner)
  }

  return { onUpdate }
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
  let data = fixupData(inData)

  let ui = newUI(data, gameId, name, colour, processUpdate, processTurn)
  window.ui = ui

  ui.addComponent(makeLog)
  ui.addComponent(makeNotifier)
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
  ui.addComponent(makeDebt)
  ui.addComponent(makeWindicator)

  netState = connect(ui, { game: gameId, name, colour })
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
  document.querySelector('.map').append(mapObject)

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

// unused

function doSay() {
  let msg = prompt('Say what?')
  if (!msg) return
  netState.send('text', msg)
}

function scrollingMap() {
  let map = document.querySelector('.map')
  setupScrolling(map, map)
  // let svg = map.querySelector('object').contentDocument.rootElement
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
