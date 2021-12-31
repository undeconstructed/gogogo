
function showMessage(msg) {
  let div = document.querySelector('.makegame')
  let msgDiv = div.querySelector('.message')
  msgDiv.textContent = msg
  div.setAttribute('show', 'message')
}

function doCreate(options, players) {
  let js = JSON.stringify({ 'type': 'rummy', options, players })
  fetch('/api/games', { method: 'POST', body: js }).
    then(rez => {
      if (!rez.ok) {
        rez.json().then(j => {
          showMessage(j.error.message)
        })
      } else {
        rez.json().then(j => {
          doOnCreate(j)
        })
      }
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
    let th0 = document.createElement('th')
    th0.textContent = k
    let td0 = document.createElement('td')
    let a = document.createElement('a')
    let link = `${window.location.origin}/play/rummy/?c=${c}`
    a.href = link
    a.textContent = link
    td0.append(a)
    tr.append(th0, td0)
    tbody.append(tr)
  }
  div.setAttribute('show', 'output')
}

function main() {
  let div = document.querySelector('.makegame')
  let inpDiv = div.querySelector('.input')
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
      players.push({ name: n })
    }
    inpDiv.style.display = 'none'
    showMessage('... working ...')
    doCreate({}, players)
  })

  div.setAttribute('show', 'input')
}

document.addEventListener('DOMContentLoaded', main)
