# prometheus-nrpe-exporter
Prometheus metrics endpoint running Nagios NRPE plugins

## Purpose

This program is intended to be used as a replacement for the NRPE daemon on
machines, providing an endpoint that Prometheus can scrape for metrics that
includes the return code and output of any configured NRPE plugins.

## Config

An example yaml config file is included with the repo.  Each check needs to
contain a name, and the command line it is to run.

## Usage

Start the daemon by whichever means is appropriate, as whichever user you would
use for the NRPE daemon (or create it if needs be).

Go to the url http://hostname:port/metrics for Prometheus metrics, and
http://hostname:port/check_name for the status output text of the individual
checks.
