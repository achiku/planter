#! /usr/bin/env bash

(
  cd ..
  go install
)

planter "postgres://planter@${PGHOST:-localhost}/planter?sslmode=disable" --output=example_gen.uml
java -jar plantuml.jar -verbose example_gen.uml
