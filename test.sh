#!/bin/bash

docker build --rm -t test .
docker run -it --rm test
