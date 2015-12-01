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


## Running

Apparatchick is meant to run as a Docker container. It will listen on the port 8080 for the HTTP requests and use volume with path '/applications' to store state.

The recomended way of startin Aparatchick is:

```bash
docker run -v /var/run/docker.sock:/var/run/docker.sock -v /applications:/applications -p 8080:8080 --name=apparatchik -d netice9/apparatchik:0.0.1
```

If you would like to add Basic Auth to the HTTP, set AUTH_USERNAME and AUTH_PASSWORD environment variables when starting Aparatchick:

```bash
docker run -v /var/run/docker.sock:/var/run/docker.sock -v /applications:/applications -p 8080:8080 --name=apparatchik -e AUTH_USERNAME=admin -e AUTH_PASSWORD=adminspassword -d netice9/apparatchik:0.0.1
```

## Web interface

Apparatchick comes with a very useful Web interface written in React.js (yes, it is an JS application and won't work without a modern browser).

The web interface will allow you to create new applications (by uploading an application descriptor - see API for details), stop (delete) a running application and monitor a running application. To get to the web interface just enter the url pointing to the HTTP interface of Apparatchick in your browser (e.g. http://your.docker.host.com:8080/)

## Terminology

### Application
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


## RESTful interface

Apparatchik offers a very simple API for starting, monitoring and removing appllications.

### Endpoints

#### `GET /api/v1.0/applications`
Returns an JSON array containing names of all currently running or starting applications

#### `PUT /api/v1.0/applications/:applicationName`

Creates an application. The body of the request is an application descriptor.
Application descriptor is be a JSON object describing the application in the following format:

| Name      | Description                                                                                |
| ----------| -----------                                                                                |
| goals     | Object describing all goals. Key is the name of the goal and value is the goal description |
| main_goal | Name of the main goal for the application. Apparatchick will try to start this goal, but will first make sure that linked goals and run_after goals are started first |

Each goal description has more or less structure of service description of Docker-Compose, with following additional parameters:

| Name      | Description                                                                                |
| ----------| -----------                                                                                |
| auth_config  | Authentication used to download Image from the registry. Not needed when image is in a public repository. JSON object with keys "username" and "password"  |
| smart_restart | Boolean value. When true, Apparatchik will restart the goal if it exits with a code != 0. Also all goals depending on this goal will be started |
| run_after | List of goal names that need to succesfully terminate before this goal can start. This is extension of the service model of Docker Compose to allow for temporal execution dependency of things like set up scripts |

For example, an application descriptor for a Rails application that uses Postgres DB and needs to run db:setup and db:migrate command before starting would look like this:

```json
{
  "goals": {
    "dbsetup": {
      "auth_config": {
        "username": "your_repo_username",
        "password": "youre_repo_password"
      },
      "image": "your-repo.com:port/user/rails_image_name:tag",
      "command": [
        "rake",
        "db:setup"
      ],
      "links": [
        "pg"
      ],
      "smart_restart": true
    },
    "migrate": {
      "auth_config": {
        "username": "your_repo_username",
        "password": "youre_repo_password"
      },
      "image": "your-repo.com:port/user/rails_image_name:tag",
      "command": [
        "rake",
        "db:migrate"
      ],
      "run_after": [
        "dbsetup"
      ],
      "links": [
        "pg"
      ],
      "smart_restart": true
    },
    "pg": {
      "image": "postgres:9.4.5",
      "environment": {
        "POSTGRES_USER": "database_username",
        "POSTGRES_PASSWORD": "database_password"
      },
      "smart_restart": true
    },
    "rails": {
      "auth_config": {
        "username": "your_repo_username",
        "password": "youre_repo_password"
      },
      "image": "your-repo.com:port/user/rails_image_name:tag",
      "command": [
        "rails",
        "server",
        "-b",
        "0.0.0.0"
      ],
      "run_after": [
        "migrate"
      ],
      "links": [
        "pg"
      ],
      "ports": [
        "3000:3000"
      ],
      "smart_restart": true
    }
  },
  "main_goal": "rails"
}
```

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

The object always has following properties:

| Name      | Description                               |
| ----------| -----------                               |
| name      | Name of the application                   |
| goals     | Object describing state of each goal      |
| main_goal | Name of the main goal for the application |

Each goal state describing object can have following properties:

| Name      | Description                                                                   |
| ----------| -----------                                                                   |
| name      | Name of the goal                                                              |
| status    | Object describing state of each goal                                          |
| exit_code | Exit code of the goal process. Only set if the status is terminated or failed |


...

