#!/bin/bash

imageTage=soorena776/testing-wh:v9

docker build . --tag=$imageTage
docker push $imageTage