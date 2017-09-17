#!/bin/bash

(
  cd ..
  go install
)

planter postgres://planter@localhost/planter?sslmode=disable --output=example_gen.uml
java -jar plantuml.jar -verbose example_gen.uml
