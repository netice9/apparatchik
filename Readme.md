# Apparatchik
> A man not of grand plans, but of a hundred carefully executed details.
--

## What is Apparatchik?

Apparatchik is a Docker-Compose-like RESTful service running as a Docker container and so much more.
Some of the features of Apparatchik are:

- RESTful interface to deploy (PUT), inspect (GET), and remove (DELETE) applications.
- User friendy browser UI (React.js application) utilizing the RESTful interface, making deployment and management of applications a breeze.
- Extension of Docker-Compose model with **run_after** dependency model making it possible to have temporal dependency in the order of running containers.
- Tracking of container's recent Memory and CPU statistics.
- Smart restarting of all components depending on a crashed component - no need to check /etc/hosts for change of IP addresses of services.
- Easy to run as a Docker container.
- Implemented completely in Golang on the server side to minimize resource consumption.

## Why should I use Apparatchik?

Imagine deploying your typical Rails or Node web application that contains multiple dependencies such as Database, Memcached or ElasticSearch in production by just PUTing one JSON file to a RESTful endpoint. There is no need to imagine that anymore - you can do it by using Aparatchik.

## RESTful interface

Apparatchik offers a very simple API for starting, monitoring and removing appllications.

### Terminology


#### Application
Application is self contained unit of software. Application usually has few to none external dependencies and is able to run on its own, independent on where it is deployed. Each Application consists of at least one goal, but usually several goals that form an directed graph towards the main goal of the application.

A simple example of an application would be a Ruby on Rails goal that runs after database create and migrate goals and depends on a PostgreSQL database.

### Goal
Goal is the atomic part of an application. Goal represents one Docker container. In Docker-Compose terms, goal is a Service.
Goals can depend on each other either by being linked or by declaring **run_after** dependency to another goal.
Goals are described in a very similar structure as in Docker-Compose with the difference that the structure encoded as JSON object.

### Main Goal
Main goals is the goal that should be either running or succesfully executed (exited with the status 0) for the application to be running.
Once an application is created, Apparatchik will try to execute the main goal by starting it.
If the main goal depends on other goals, Apparatchick will recursively either start (in the case of a linked Goal) or execute and wait for a successful termination (**run_after** dependencies) before the main Goal is being started.

### Endpoints

#### `GET /api/v1.0/applications`
Returns an JSON array containing names of all currently running or starting applications

#### `GET /api/v1.0/applications/:applicationName`
Returns an JSON object describing the state of an application. The object has a following format:

```json
{
  "name": "app1",
  "goals": {
    "dbsetup": {
      "name": "dbsetup",
      "status": "terminated",
      "exit_code": 0
    },
    "migrate": {
      "name": "migrate",
      "status": "terminated",
      "exit_code": 0
    },
    "pg": {
      "name": "pg",
      "status": "running"
    },
    "rails": {
      "name": "rails",
      "status": "running"
    }
  },
  "main_goal": "rails"
}
```
| Name      | Description                               |
| ----------| -----------                               |
| name      | Name of the application                   |
| goals     | Object describing state of each goal      |
| main_goal | Name of the main goal for the application |




