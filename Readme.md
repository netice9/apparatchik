# Apparatchik
> A man not of grand plans, but of a hundred carefully executed details.
--

## What is Apparatchik?

Apparatchick is a Docker-Compose-like RESTful service running as a Docker container and so much more.
Some of the features of Apparatchick are:

- RESTful interface to deploy (PUT), inspect (GET), and remove (DELETE) applications.
- User friendy browser UI (React.js application) untilizing the RESTful interface, making deployment and management of applications a breeze.
- Extension of Docker-Compose model with **run_after** dependency model making it possible to have temporal dependency in the order of running containers.
- Implemented completely in Golang on the server side to minimize resource consumption.

