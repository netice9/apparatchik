var Tooltip = require('react-bootstrap').Tooltip;
var OverlayTrigger = require('react-bootstrap').OverlayTrigger;
var React = require('react');
var ReactDOM = require('react-dom');
var _ = require('lodash')
var jq = require('jquery');


module.exports = React.createClass({
  shouldComponentUpdate: function(nextProps, nextState){
    return nextState.log !== this.state.log
  },
  getInitialState: function() {
    return {log: 'Loading Log ...'}
  },
  componentDidUpdate: function() {
    var node = ReactDOM.findDOMNode(this);
    node.scrollTop = node.scrollHeight;
  },
  updateLogs: function() {
    jq.get("/api/v1.0/applications/"+this.props.applicationName+"/goals/"+this.props.goalName+"/logs", function(result) {
      this.setState({log: result});
    }.bind(this));
  },
  componentDidMount: function() {
    this.updateLogs();
    this.timer = setInterval(this.updateLogs, 1000);
  },
  componentWillUnmount: function() {
    clearTimeout(this.timer);
  },
  render: function() {
    var lines =_.map(this.state.log.split("\n"), function(line, index) {
      return <div key={index}
      style={{
        whiteSpace: 'nowrap',
      }}>{line}</div>
    } )
    return (
      <div style={{
        overflowY: 'auto',
        overflowX: 'auto',
        maxHeight: '420px',
        fontFamily: "Courier New",
        monospace: true
      }}>
       <div>
          { lines }
       </div>
      </div>
    )
  }
});