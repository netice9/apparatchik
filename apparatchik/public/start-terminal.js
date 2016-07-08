
function startTerminal() {
  var term,
      protocol,
      socketURL,
      socket,
  		path;
  var terminalContainer = document.getElementById('terminal-container');
  while (terminalContainer.children.length) {
  	terminalContainer.removeChild(terminalContainer.children[0]);
  }
  path = terminalContainer.dataset.apiPath
  protocol = (location.protocol === 'https:') ? 'wss://' : 'ws://';
  socketURL = protocol + location.hostname + ((location.port) ? (':' + location.port) : '') + path;
  socket = new WebSocket(socketURL);
  term = new Terminal({
  	cursorBlink: true
  });
  term.open(terminalContainer);

  socket.onopen = function() {
    term.attach(socket);
    term._initialized = true;
  }
}
