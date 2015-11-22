var React = require('react');
var Panel = require('react-bootstrap').Panel;
var Button = require('react-bootstrap').Button;
var Glyphicon = require('react-bootstrap').Glyphicon;
var Modal = require('react-bootstrap').Modal;
var Table = require('react-bootstrap').Table;
var jq = require('jquery');
var update = require('react-addons-update');
var _ = require('lodash')
var Terminal = require('./terminal')
// var ReactDOM = require('react-dom');


var Tooltip = require('react-bootstrap').Tooltip;
var OverlayTrigger = require('react-bootstrap').OverlayTrigger;

var Popup = React.createClass({
  getInitialState: function() {
    return {show: false}
  },
  close() {
    this.setState({show: false});
  },
  open() {
    document.activeElement.blur();
    this.setState({show: true});
  },
  render: function() {

   var tooltip = <Tooltip id="tt">{this.props.tooltip}</Tooltip>;

   var children = this.open ? this.props.children : []
   return (
    <span>
      <OverlayTrigger overlay={tooltip}>
        <Button onClick={this.open} id={this.props.id}>{this.props.name}</Button>
      </OverlayTrigger>
      <Modal show={this.state.show} onHide={this.close} bsSize="large" keyboard={false}>
        <Modal.Header closeButton>
          <Modal.Title>{this.props.title}</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          {children}
        </Modal.Body>
        <Modal.Footer>
          <Button onClick={this.close}>Close</Button>
        </Modal.Footer>
      </Modal>
    </span>
    )
  }
});

var ApplicationLog = require('./application_log')
var ApplicationStats = require('./application_stats')
var ApplicationTransitionLog = require('./application_transition_log')

var Application = React.createClass({
  getInitialState: function() {
    return { application: {}, showModal: false };

  },
  delete() {
    var that = this;
    jq.ajax({
      url: "/api/v1.0/applications/"+this.props.params.applicationName,
      type: 'DELETE',
      success: function() {
        that.props.history.pushState(null,'/');
      },
      error: function(e) {
        alert(e);
      }
    });
  },
  close() {
    this.setState(update(this.state, {showModal: {$set: false}} ));
  },

  open() {
    this.setState(update(this.state, {showModal: {$set: true}} ));
  },
  updateApplication: function() {
    jq.get("/api/v1.0/applications/"+this.props.params.applicationName, function(result) {
      this.setState(update(this.state, {application: {$set: result}}));
    }.bind(this));
  },
  componentDidMount: function() {
    this.updateApplication();
    this.timer = setInterval(this.updateApplication, 1000);
  },
  componentWillUnmount: function() {
    clearTimeout(this.timer);
  },
  render: function() {


    var goals = _.map(this.state.application.goals, function(goal) {
      var applicationName = this.props.params.applicationName;
      return (
        <tr key={goal.name}>
          <td className="goal-name">{goal.name}</td>
          <td>{goal.status}</td>
          <td>
            <Popup title={"Output Log for "+applicationName+" > "+goal.name} tooltip="Display output log" name={<Glyphicon glyph="paste" />} id={goal.name + "_logs"}><ApplicationLog applicationName={applicationName} goalName={goal.name}/></Popup>
            <Popup title={"Stats for "+applicationName+" > "+goal.name} tooltip="Display stats graph" name={<Glyphicon glyph="equalizer" />} id={goal.name + "_stats"}><ApplicationStats applicationName={applicationName} goalName={goal.name}/></Popup>
            <Popup title={"State Tranistion Log for "+applicationName+" > "+goal.name} tooltip="Display state transition log" name={<Glyphicon glyph="transfer" />} id={goal.name + "_transitions"}><ApplicationTransitionLog applicationName={applicationName} goalName={goal.name}/></Popup>
            <Popup title={"Terminal for "+applicationName+" > "+goal.name} tooltip="Open interactive terminal" name={<Glyphicon glyph="console" />} id={goal.name + "_terminal"}><Terminal applicationName={applicationName} goalName={goal.name}/></Popup>
          </td>
        </tr>
        )
    }.bind(this));

    if (!this.state.application) {
      return(<div>Loading ...</div>)
    }

    return(
      <div className="container-fluid">
        <div className="row">
          <div className="col-md-4">
            <Panel header="General Information">
              <dl>
                <dt>Name</dt>
                <dd id="application_name">{this.state.application.name}</dd>
              </dl>
              <dl>
                <dt>Main Goal</dt>
                <dd id="main_goal">{this.state.application.main_goal}</dd>
              </dl>
              <Button bsStyle="danger" onClick={this.open}><Glyphicon glyph="remove" /> Delete</Button>
              <Modal show={this.state.showModal} onHide={this.close}>
                <Modal.Header closeButton>
                  <Modal.Title>Delete Application {this.state.application.name}?</Modal.Title>
                </Modal.Header>
                <Modal.Body>
                  <h4>Are you sure?</h4>
                  <p>Deleting application will stop all goals and remove all containers.</p>
                  <hr />
                </Modal.Body>
                <Modal.Footer>
                  <Button bsStyle="danger" onClick={this.delete} id="delete_confirm">Delete</Button>
                  <Button onClick={this.close}>Cancel</Button>
                </Modal.Footer>
              </Modal>

            </Panel>
          </div>
        </div>
        <div className="row">
          <div className="col-md-6">
            <Panel header="Goals">
              <Table striped bordered condensed hover>
               <thead>
                <tr>
                  <th>Name</th>
                  <th>State</th>
                  <th>Actions</th>
                </tr>
               </thead>
               <tbody>
                {goals}
               </tbody>
              </Table>
            </Panel>
          </div>
        </div>
      </div>
    )
  }
})

module.exports = Application