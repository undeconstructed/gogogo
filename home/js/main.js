
function doCreate(options, players) {
  let js = JSON.stringify({ options, players })
  fetch('/api/games', { method: 'POST', body: js }).
    then(rez => rez.json()).
    then(data => {
      doOnCreate(data)
    })
}

function doOnCreate(msg) {
  let div = document.querySelector('.makegame')
  let outDiv = div.querySelector('.output')
  let tbody = outDiv.querySelector('tbody')
  tbody.replaceChildren()
  for (let k in msg.players) {
    let c = msg.players[k]
    let tr = document.createElement('tr')
    let td0 = document.createElement('td')
    td0.textContent = k
    let td1 = document.createElement('td')
    let a = document.createElement('a')
    let link = `${window.location.origin}/play/?c=${c}`
    a.href = link
    a.textContent = link
    td1.append(a)
    tr.append(td0, td1)
    tbody.append(tr)
  }
  outDiv.style.display = 'block'
}

function main() {
  let div = document.querySelector('.makegame')
  let inpDiv = div.querySelector('.input')
  let outDiv = div.querySelector('.output')
  let form = div.querySelector('.input form')

  let playersDiv = form.querySelector('.players')
  let playerTmpl = document.querySelector('#playerline').content.firstElementChild

  let addPlayerLine = () => {
    let n = playerTmpl.cloneNode(true)
    n.querySelector('.removeplayer').addEventListener('click', _e => {
      n.remove()
    })
    playersDiv.append(n)
  }

  addPlayerLine()

  let addPlayerButton = form.querySelector('.addplayer')
  addPlayerButton.addEventListener('click', _e => {
    addPlayerLine()
  })

  form.addEventListener('submit', e => {
    e.preventDefault()
    let players = []
    for (let p of playersDiv.querySelectorAll('.player')) {
      let n = p.querySelector('input').value
      let c = p.querySelector('select').value
      players.push({ name: n, colour: c })
    }
    inpDiv.style.display = 'none'
    doCreate({}, players)
  })
}

document.addEventListener('DOMContentLoaded', main)
