# Apparatchik
> A man not of grand plans, but of a hundred carefully executed details.
--

## What is Apparatchik?

Apparatchick is a Docker-Compose-like RESTful service running as a Docker container and so much more.
Some of the features of Apparatchick are:

- RESTful interface to deploy (PUT), inspect (GET), and remove (DELETE) applications.
- User friendy browser UI (React.js application) untilizing the RESTful interface, making deployment and management of applications a breeze.
- Extension of Docker-Compose model with **run_after** dependency model making it possible to have temporal dependency in the order of running containers.
- Adds smart restarting of all components depending on a crashed component - no need to check /etc/hosts for change of IP addresses of services.
- Easy to run as a Docker container.
- Implemented completely in Golang on the server side to minimize resource consumption.

## Why should I use Apparatchik?

Imagine deploying your typical Rails or Node web application that contains multiple dependencies such as Database, Memcached or ElasticSearch in production by just PUTing one JSON file to a RESTful endpoint. There is no need to imagine that anymore - you can do it by using Aparatchick.

