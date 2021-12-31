
import { newUI, connect } from '/common/js/game.js'

// net stuff

let netState = null

function processUpdate(u) {
  return u
}

// game utils

// ui components

// game setup

function setup(inData, ccode) {
  let ui = newUI({}, processUpdate)
  window.ui = ui

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

  setup({}, ccode)

  window.cheat = function(cmd) {
    netState.doRequest('play', { command: 'cheat', options: cmd }, console.log)
  }
}

document.addEventListener('DOMContentLoaded', main)
