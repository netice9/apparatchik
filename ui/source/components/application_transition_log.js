var React = require('react');
var _ = require('lodash')
var jq = require('jquery');
var vis = require("vis");
var Panel = require('react-bootstrap').Panel;


module.exports = React.createClass({
  updateTransitionTimeline: function() {
    jq.get("/applications/"+this.props.applicationName+"/"+this.props.goalName+"/transition_log", {since:  this.last_time} , function(result) {


      var data =[];

      _.reduce(result, function(last, current) {
        data.push({ id: Date.parse(last.time), start: last.time, end: Date.parse(current.time)-100, content: last.status});
        return current;
      });

      if (result.length > 0) {
        var last = result[result.length - 1];
        data.push({ id: Date.parse(last.time), start: last.time, content: last.status})
      }

      this.timelineData.update(data, "api");
      this.timelineData.flush();
      this.timeline.fit({animation: false});

    }.bind(this));
  },
  componentDidMount: function() {
    this.timelineData = new vis.DataSet({queue: true});
    this.createVis();
    this.updateTransitionTimeline();
    this.timer = setInterval(this.updateTransitionTimeline, 1000);
  },
  componentWillUnmount: function() {
    clearTimeout(this.timer);
    this.timeline.destroy();
  },
  createVis: function() {
    var options = {
      width:  '100%',
      height:  '100%',
      moveable: false,
      stack: true,
    };

    this.timeline = new vis.Timeline(document.getElementById('timeline'), this.timelineData, options);

  },
  render: function() {
    return(
      <div id="timeline" style={ {height: "200px"} }/>
    )
  }
});