var React = require('react');
var _ = require('lodash')
var jq = require('jquery');
var vis = require("vis");
var Panel = require('react-bootstrap').Panel;


module.exports = React.createClass({
  updateStats: function() {
    jq.get("/applications/"+this.props.applicationName+"/"+this.props.goalName+"/stats", {since:  this.last_time} , function(result) {
      this.cpuData.update(_.map(result.cpu_stats, function(stat) { return { id: Date.parse(stat.time), x: stat.time, y: stat.value /10000000 } } ), "api");
      var toRemove = this.cpuData.length - 120 + 1
      for (var i=0; i<toRemove; i++) {
        this.cpuData.remove(this.cpuData.getIds()[0]);
      }
      this.cpuData.flush();
      this.cpuGraph.fit({animation: false});

      this.memData.update(_.map(result.mem_stats, function(stat) { return { id: Date.parse(stat.time), x: stat.time, y: stat.value /(1024*1024) } } ), "api");
      var toRemove = this.memData.length - 120 + 1
      for (var i=0; i<toRemove; i++) {
        this.memData.remove(this.memData.getIds()[0]);
      }
      this.memData.flush();
      this.memGraph.fit({animation: false});

      if (result.cpu_stats.length > 0) {
        this.last_time = result.cpu_stats[result.cpu_stats.length-1].time
      }
    }.bind(this));
  },
  componentDidMount: function() {
    this.cpuData = new vis.DataSet({queue: true});
    this.memData = new vis.DataSet({queue: true});
    this.createVis();
    this.updateStats();
    this.timer = setInterval(this.updateStats, 1000);
  },
  componentWillUnmount: function() {
    clearTimeout(this.timer);
    this.cpuGraph.destroy();
    this.memGraph.destroy();
  },
  createVis: function() {
    var options = {
      width:  '100%',
      height:  '100%',
      min: 0,
      moveable: true,
      zoomable: true,
      drawPoints: false,
      dataAxis: {
        left: {
          format: function(value) {
            return value.toFixed(2);
          },
          range: {
            min: 0
          }
        }

      }

    };

    this.cpuGraph = new vis.Graph2d(document.getElementById('cpu_stats_graph'), this.cpuData, options);
    this.memGraph = new vis.Graph2d(document.getElementById('mem_stats_graph'), this.memData, options);

  },
  render: function() {
    return(
      <div>
        <Panel header="CPU Stats (%)">
          <div id="cpu_stats_graph" style={ {height: "200px"} }/>
        </Panel>
        <Panel header="Memory Stats (MB)">
          <div id="mem_stats_graph" style={ {height: "200px"} }/>
        </Panel>
      </div>
    )
  }
});