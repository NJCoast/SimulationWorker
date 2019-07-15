# Simulation Queue Master

[![GoDoc](https://godoc.org/github.com/NJCoast/SimulationWorker?status.svg)](https://godoc.org/github.com/NJCoast/SimulationWorker)
[![Go Report Card](https://goreportcard.com/badge/github.com/NJCoast/SimulationWorker)](https://goreportcard.com/report/github.com/NJCoast/SimulationWorker)
[![TravisCI](https://travis-ci.org/NJCoast/SimulationWorker.svg?branch=master)](https://travis-ci.org/NJCoast/SimulationWorker)

## Installation

The SimulationWorker tool is designed to be run as a service and deployed in a containerized environment. This repository contains a [Dockerfile](https://docs.docker.com/engine/reference/builder/) that can be used to build and deploy the tool using docker. To build the container:

```bash
docker build -t SimulationWorker .
```

## Development

The Simulation Queue Worker is written in [go](https://golang.org/) for deployment as a microservice. Code documentation is autogenerated by the [GoDoc](https://godoc.org/) documentation system. The code is setup to use continious integration using [TravisCI](https://travis-ci.org)

## Base Image Build

The base image is a compiled and containerized form of our Matlab model. The following components are built with Matlab's Application Compiler.

* ObtainingParametersCrossingPoint

  This is used to generate an input file from a hurricane's track.

* WebCentralAnalysis

  This executes the hurricane, nor'easter and wind models and writes the results to a local storage.

Once these applications have been built, they need to be installed, with the runtime, to a known place. The files from these applications are then copied to the ```model/``` directory and the runtime files need to be copied to the ```model/runtime``` directory. They are built with the following:

```bash
docker build -t model:runtime model/runtime
docker build -t model:latest model/
```
