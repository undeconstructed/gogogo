
import { newUI, connect, promoteCustom } from '/common/js/game.js'

// net stuff

let netState = null

function processUpdate(u) {
  for (let pl of u.players) {
    let gp = u.global.players[pl.name]
    if (gp) {
      for (let x in gp) {
        pl[x] = gp[x]
      }
    }

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

// game utils

function makeByLine(data, modes) {
  let ms = []
  for (let m of modes) {
    ms.push(data.modes[m])
  }
  return ms.join('/')
}

function buttonCallback0(div, then) {
  return (e, r) => {
    if (e) {
      alert(e.message)
      div.classList.add('open')
      return
    }
    then && then(r)
  }
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

function makeStatusBar(_data, up) {
  let aboutGame = document.querySelector('.aboutgame')

  ;(() => {
    aboutGame.querySelector('.msg').addEventListener('click', _e => {
      up.send({ do: 'chat' })
    })
  })()

  let doUpdatePlayers = (players, playing) => {
    let playersDiv = aboutGame.querySelector('.players > div')
    playersDiv.replaceChildren()
    for (let name in players) {
      let pl = players[name]

      if (pl.name == playing) {
        let sp = aboutGame.querySelector('.now')
        sp.style.borderColor = pl.colour
        let sn = sp.querySelector('.name')
        sn.textContent = pl.name
      }

      let div = document.createElement('div')
      div.classList.add('aplayer')
      div.style.borderColor = pl.colour || 'transparent'
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
    let me = aboutGame.querySelector('.me')
    me.style.borderColor = s.me.colour
    let sn = me.querySelector('.name')
    sn.textContent = s.me.name
  }

  return { onUpdate }
}

function makeMap(data, up) {
  let marks = {}

  let svg = document.querySelector('.map > object').contentDocument
  let dotsLayer = svg.querySelector('#dotslayer')
  let playersLayer = svg.querySelector('#playerslayer')

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
        dotsLayer.append(ndot)
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
        ndot.addEventListener('click', _e => { console.log(pointId) })
        dotsLayer.append(ndot)
      }
    }

    for (let pointId in data.dots) {
      drawPoint(pointId, data.dots[pointId])
    }
  })()

  let makeMark = (number, colour, dot) => {
    if (!colour) {
      return
    }

    let prev = marks[colour]
    if (prev) {
      prev.remove()
    }

    let point = data.dots[dot]
    let [x, y] = splitDotId(dot)

    let nmarker = null

    if (point.terminal) {
      let rotate = 60*number
      let marker = svg.querySelector('#playerwedge')
      nmarker = marker.cloneNode(true)
      nmarker.id = 'player-' + colour
      nmarker.setAttributeNS(null, 'x', x-22)
      nmarker.setAttributeNS(null, 'y', y-22)
      nmarker.firstElementChild.setAttributeNS(null, 'transform', `rotate(${rotate},22,22)`)
      nmarker.firstElementChild.style.fill = colour
    } else {
      let marker = svg.querySelector('#playerring')
      nmarker = marker.cloneNode(true)
      nmarker.id = 'player-' + colour
      nmarker.setAttributeNS(null, 'cx', x)
      nmarker.setAttributeNS(null, 'cy', y)
      nmarker.style.stroke = colour
    }

    marks[colour] = nmarker
    playersLayer.append(nmarker)
  }

  let makeMarks = (players, playing) => {
    for (let name in players) {
      let pl = players[name]
      makeMark(pl.number, pl.colour, pl.dot)
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
  let freeticket = null

  let isFree = (from, to, by) => {
    let match = (value, filter) => {
      return !filter || filter === '*' || filter === value
    }
    if (freeticket) {
      return match(from, freeticket.from) && match(to, freeticket.to) && match(by, freeticket.by)
    }
    return false
  }

  let doSetupList = () => {
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

        if (isFree(placeId, destId, modeId)) {
          fare = 0
        }

        let td1 = document.createElement('td')
        td1.classList.add('place')
        td1.append(dest)
        tr.append(td1)

        let td2 = document.createElement('td')
        td2.append(mode)
        tr.append(td2)

        let td3 = document.createElement('td')
        td3.classList.add('fare')
        td3.append(`£${fare*stRate/100}`)
        tr.append(td3)
        let td4 = document.createElement('td')
        td4.classList.add('fare')
        td4.append(`${fare*loRate/100}`)
        tr.append(td4)

        tr.addEventListener('click', _e => {
          if (freeticket) {
            freeticket.cb(placeId, destId, modeId)
          } else {
            let cb = (e, _r) => {
              if (e) { alert(e.message); return; }
              up.send({ do: 'notify', msg: 'you have bought a ticket' })
            }
            netState.doRequest('play', { command: `buyticket:${placeId}:${destId}:${modeId}` }, cb)
          }
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
  }

  let doOpen = (placeId) => {
    ele.classList.add('open')
    if (placeId) {
      let p = ele.querySelector(`.${placeId}`)
      p.scrollIntoView({ behavior: 'smooth' })
    }
  }

  let doOpenFree = (placeId, filter) => {
    freeticket = filter
    doSetupList()
    doOpen(placeId)
  }

  let doClose = () => {
    ele.classList.remove('open')
    if (freeticket) {
      freeticket = null
      doSetupList()
    }
  }

  let onCommand = c => {
    if (c.do === 'showprices') {
      doOpen(c.at)
    } else if (c.do === 'freeticket') {
      // TODO - need to have current location here
      doOpenFree(c.at, c)
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

  ;(() => {
    doSetupList({})
    ele.addEventListener('click', doClose)
  })()

  return { onCommand, onTurn }
}

function makeRateList(data) {
  let div = document.querySelector('.rates')

  ;(() => {
    let denoms = [ 1, 2, 5, 10, 20, 50 ]
    let groups = {}

    for (let cId in data.currencies) {
      let c = data.currencies[cId]
      let g = groups[c.rate] || []
      g.push(c.name)
      groups[c.rate] = g
    }

    let thead = div.querySelector('thead')
    thead.replaceChildren()
    {
      let tr = document.createElement('tr')
      for (let gId in groups) {
        let g = groups[gId]
        let th = document.createElement('th')
        th.append(g.join(' '))
        tr.append(th)
      }
      thead.append(tr)
    }

    let tbody = div.querySelector('tbody')
    tbody.replaceChildren()
    for (let d of denoms) {
      let tr = document.createElement('tr')
      for (let g in groups) {
        let td = document.createElement('td')
        td.textContent = (d * g).toLocaleString()
        tr.append(td)
      }
      tbody.append(tr)
    }
  })()

  return {}
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

function makeLuckStack(data, up) {
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

  let makeDoFreeTicket = (id) => {
    return (from, to, by) => {
      let cb = (e, _r) => { if (e) { alert(e.message); return; } }
      let options = `${from}:${to}:${by}`
      netState.doRequest('play', { command: 'useluck:'+id+':'+options }, cb)
    }
  }

  let doUse = (id, card) => {
    if (card.ui === 'prompt') {
      let options = prompt('options (or none)')
      if (options == null) {
        return
      }
      let cb = (e, _r) => { if (e) { alert(e.message); return; } }
      netState.doRequest('play', { command: 'useluck:'+id+':'+options }, cb)
    } else if (card.ui === 'freeticketfixed') {
      let options = card.code.substring(11)
      let cb = (e, _r) => { if (e) { alert(e.message); return; } }
      netState.doRequest('play', { command: 'useluck:'+id+':'+options }, cb)
    } else if (card.ui === 'freeticketchoice') {
      let [_x, from, to, by] = card.code.split(':')
      up.send({ do: 'freeticket', from, to, by, cb: makeDoFreeTicket(id) })
    } else {
      let cb = (e, _r) => { if (e) { alert(e.message); return; } }
      netState.doRequest('play', { command: 'useluck:'+id+':' }, cb)
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
          doUse(luckId, luckData)
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
        let price = data.settings.souvenirPrice * currency.rate / 100
        div.querySelector('.price').textContent = '' + price + ' ' + currency.name
        for (let bar of div.querySelectorAll('.bar')) {
          bar.style.backgroundColor = currency.colour
        }
        div.querySelector('button.declare').addEventListener('click', e => {
          e.stopPropagation()
          let cb = (e, _r) => { if (e) { alert(e.message); return; } }
          netState.doRequest('play', { command: 'declare:'+placeId }, cb)
          doClose()
        })
        div.querySelector('button.pawn').addEventListener('click', e => {
          e.stopPropagation()
          let cb = (e, _r) => { if (e) { alert(e.message); return; } }
          netState.doRequest('play', { command: 'pawnsouvenir:'+placeId }, cb)
          doClose()
        })
        div.querySelector('button.sell').addEventListener('click', e => {
          e.stopPropagation()
          let cb = (e, _r) => { if (e) { alert(e.message); return; } }
          netState.doRequest('play', { command: 'sellsouvenir:'+placeId }, cb)
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
    stack.classList.toggle('canpawn', t.hasCan('pawnsouvenir:*'))
    stack.classList.toggle('cansell', t.hasCan('sellsouvenir:*'))
  }

  return { onUpdate, onTurn }
}

function makeAutoButtons(data, up) {
  let buttonBox = document.querySelector('.actions')

  let skip = [
    'airlift',
    'buysouvenir',
    'buyticket',
    'changemoney',
    'declare',
    'dicemove',
    'end',
    'gamble',
    'ignorerisk',
    'obeyrisk',
    'pawnsouvenir',
    'pay',
    'paycustoms',
    `quarantine`,
    'redeemsouvenir',
    'sellsouvenir',
    'stop',
    'takeluck',
    'takerisk',
    'useluck'
  ]
  // skip = []

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

      button.classList.add('text')
      button.append(cmd)

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
      let cb = buttonCallback0(div)
      netState.doRequest('play', { command: 'stop' }, cb)
      div.classList.remove('open')
    })
  })()

  let onTurn = t => {
    let canStop = t.hasCan('stop')
    let onMap = t.onmap
    div.classList.toggle('onmap', onMap)
    div.classList.toggle('open', canStop)
  }

  return { onTurn }
}

function makeDiceButton(_data, up) {
  let div = document.querySelector('.dicebutton')

  ;(() => {
    div.addEventListener('click', _e => {
      let cb = buttonCallback0(div, r => up.send({ do: 'notify', msg: 'you rolled a ' + r.message}))
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
      let cb = buttonCallback0(div, r => up.send({ do: 'notify', msg: 'you gambled and ' + r.message}))
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

function makeCustomsButton(_data, up) {
  let div = document.querySelector('.customsbutton')
  let cmd = ''

  ;(() => {
    div.addEventListener('click', _e => {
      let cb = buttonCallback0(div)
      netState.doRequest('play', { command: cmd }, cb)
      div.classList.remove('open')
    })
  })()

  let onTurn = t => {
    cmd = ''
    let mustDeclare = t.hasMust('declare:*')
    let mustPay = t.hasMust('paycustoms')
    if (mustDeclare) {
      cmd = 'declare:none'
    } else if (mustPay) {
      cmd = 'paycustoms'
    }
    div.classList.toggle('open', cmd !== '')
  }

  return { onTurn }
}

function makeBuySouvenirButton(_data, up) {
  let div = document.querySelector('.buysouvenirbutton')
  let cmd = ''

  ;(() => {
    div.addEventListener('click', _e => {
      let cb = buttonCallback0(div, r => up.send({ do: 'notify', msg: 'you have bought a sort of ' + r.message }))
      netState.doRequest('play', { command: cmd }, cb)
      div.classList.remove('open')
    })
  })()

  let onTurn = t => {
    cmd = ''
    for (let c of t.can) {
      if (c.startsWith('buysouvenir:')) {
        cmd = c
      }
    }
    div.classList.toggle('open', cmd !== '')
  }

  return { onTurn }
}

function makeAirliftButton(_data, up) {
  let div = document.querySelector('.airliftbutton')
  let cmd = ''

  ;(() => {
    div.addEventListener('click', _e => {
      let cb = buttonCallback0(div)
      netState.doRequest('play', { command: cmd }, cb)
      div.classList.remove('open')
    })
  })()

  let onTurn = t => {
    cmd = ''
    for (let c of t.can) {
      if (c.startsWith('airlift')) {
        cmd = c
      }
    }
    div.classList.toggle('open', cmd !== '')
  }

  return { onTurn }
}

function makeSleepButton() {
  let div = document.querySelector('.sleepbutton')
  let cmd = 'end'

  ;(() => {
    div.addEventListener('click', _e => {
      let cb = buttonCallback0(div)
      netState.doRequest('play', { command: cmd }, cb)
      div.classList.remove('open')
    })
  })()

  let onTurn = t => {
    let canEnd = t.hasCan('end')
    let canQuarantine = t.hasMust('quarantine')
    if (canQuarantine) {
      cmd = 'quarantine'
    } else {
      cmd = 'end'
    }
    canEnd = canEnd || canQuarantine
    div.classList.toggle('sick', canQuarantine)
    div.classList.toggle('open', canEnd)
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
  let body = div.querySelector('tbody')

  let onUpdate = s => {
    body.replaceChildren()
    let hasDebt = s.me.debts != null
    if (hasDebt) {
      for (let debt of s.me.debts) {
        let line = document.createElement('tr')
        let td0 = document.createElement('td')
        td0.textContent = debt.reason
        let td1 = document.createElement('td')
        td1.textContent = debt.amount
        let td2 = document.createElement('td')
        td2.textContent = debt.currency
        line.append(td0, td1, td2)
        body.append(line)
      }
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

function makeChatter() {
  let onCommand = c => {
    if (c.do === 'chat') {
      let m = prompt('send?')
      if (!m) {
        return
      }
      netState.send('text', m)
    }
  }

  return { onCommand }
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

  for (let luck of indata.lucks) {
    if (luck.retain) {
      luck.ui = 'prompt'
      let code = luck.code
      if (code.match(/^advance:\d+$/)) {
        luck.ui = null
      } else if (code.match(/^freeticket:/)) {
        if (!code.includes('*')) {
          luck.ui = 'freeticketfixed'
        } else {
          luck.ui = 'freeticketchoice'
        }
      } else if (code === 'freeinsurance' || code === 'dest' || code === 'inoculation' || code === 'immunity') {
        luck.ui = null
      }
    }
  }

  return indata
}

function setup(inData, ccode) {
  let data = fixupData(inData)

  let ui = newUI(data, processUpdate)
  window.ui = ui

  ui.addComponent(makeLog)
  ui.addComponent(makeNotifier)
  ui.addComponent(makeStartButton)
  ui.addComponent(makeStatusBar)
  ui.addComponent(makeMap)
  ui.addComponent(makeSquares)
  ui.addComponent(makePriceList)
  ui.addComponent(makeRateList)
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
  ui.addComponent(makeCustomsButton)
  ui.addComponent(makeBuySouvenirButton)
  ui.addComponent(makeAirliftButton)
  ui.addComponent(makeDebt)
  ui.addComponent(makeWindicator)
  ui.addComponent(makeChatter)

  netState = connect(ui, ccode)
}

// main()

function main() {
  let urlParams = new URLSearchParams(window.location.search)
  let ccode = urlParams.get('c')
  if (!ccode) {
    alert('missing connect code')
    return
  }

  let mapObject = document.createElement('object')
  mapObject.type = 'image/svg+xml'
  mapObject.data = 'map.svg'
  document.querySelector('.map').append(mapObject)

  mapObject.addEventListener('load', _e => {
    fetch('data.json').then(
      rez => rez.json().then(
        data => setup(data, ccode)
      ),
      err => alert('missing data: ' + err)
    )
  })

  window.cheat = function(cmd) {
    netState.doRequest('play', { command: 'cheat', options: cmd }, console.log)
  }
}

document.addEventListener('DOMContentLoaded', main)

// unused

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
