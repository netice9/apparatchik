var React = require('react');
var Terminal = require('term.js');

module.exports = React.createClass({
  componentDidMount: function() {
    var term = new Terminal({
      cols: 80,
      rows: 24,
      screenKeys: true
    });


    term.open(document.getElementById('terminal'));

    var location = window.location;

    var consoleSocket = new WebSocket("ws:"+location.host+"/api/v1.0/applications/"+this.props.applicationName+"/goals/"+this.props.goalName+"/exec")

    if (consoleSocket) {

      consoleSocket.onopen = function (event) {

        term.on('data', function(data) {
          consoleSocket.send(data);
        });
      };

      consoleSocket.onmessage = function (event) {
        term.write(event.data)
      }

      consoleSocket.onclose = function(event) {
        term.write("\r\nConnection closed!\r\n");
      }
    } else {
      term.write("\r\nCould not connect to server!\r\n");
    }


  },
  render: function() {
    return <div><div id="terminal"></div></div>
  }
})

