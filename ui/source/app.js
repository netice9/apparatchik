require('es5-shim');
var React = require('react');
var ReactDOM = require('react-dom');
var Router = require('react-router').Router;
var Route = require('react-router').Route;
var Link = require('react-router').Link;

var Button = require('react-bootstrap').Button;
var Panel = require('react-bootstrap').Panel;
var ListGroup = require('react-bootstrap').ListGroup;
var ListGroupItem = require('react-bootstrap').ListGroupItem;
var _ = require('lodash');
var jq = require('jquery');
var Application = require('./components/application');



var App = React.createClass({
  getInitialState: function() {
    return { applications: [] };
  },
  componentDidMount: function() {
    jq.get("/api/v1.0/applications", function(result) {
        this.setState({
          applications: result
        });
    }.bind(this));
  },
  render: function() {
    var apps = _.map(this.state.applications, function(name) { return <Link key={name} className="list-group-item" to={"/applications/"+name}>{name}</Link> });

    return (
      <div className="container-fluid">
        <div className="row">
         <Panel header="Active Applications">
          <ListGroup id="active_applications">
            {apps}
          </ListGroup>
          <Link key={name} className="btn btn-default" to="/new_application">New Application</Link>
         </Panel>
        </div>
      </div>
    )
  }
})

var FileInput = require('react-file-input');
var Input = require('react-bootstrap').Input;
var ButtonInput = require('react-bootstrap').ButtonInput;

var NewApplication = React.createClass({
  getInitialState: function() {
    return { disabled: true };
  },
  handleChange() {
    var file = this.refs.applicationFile.getInputDOMNode().files[0];
    var applicationName = this.refs.applicationName.getValue();
    this.setState({
      file: file,
      applicationName: applicationName,
      disabled: !(file && applicationName)
    });
  },
  createApp() {

    var reader = new FileReader();
    var that = this;
    reader.onload = function(theFile) {
      var text = reader.result;
      jq.ajax({
        url: "/api/v1.0/applications/"+that.state.applicationName,
        data: text,
        method: "PUT",
        dataType: "json",
        success: function() {
          console.log(that);
          that.props.history.pushState(null,'/');
        }
      });
    };

    reader.readAsText(this.state.file);



  },
  componentDidMount: function() {

  },
  render: function() {
    return (
      <div className="container-fluid">
        <div className="row">
          <div className="col-sm-6 col-sm-offset-3 col-lg-4 col-lg-offset-4">
           <Panel header="Upload a New Application Descriptor">
            <form>
              <Input type="text" ref="applicationName" placeholder="Application Name" label="Application Name" help="Unique name of the application" onChange={this.handleChange} />
              <Input type="file" label="File" id="file" ref="applicationFile" onChange={this.handleChange} />
              <ButtonInput value="Create" disabled={this.state.disabled} onClick={this.createApp}/>
            </form>
           </Panel>
          </div>
        </div>
      </div>
    )
  }
})



var NoMatch = React.createClass({
  render: function() {
    return <Button> Wtf? </Button>
  }
})



ReactDOM.render(
  <Router>
    <Route path="/" component={App}/>
    <Route path="/applications/:applicationName" component={Application}/>
    <Route path="/new_application" component={NewApplication}/>
  </Router>
, document.getElementById('react-application'));
