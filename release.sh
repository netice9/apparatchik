#!/bin/sh
set -e

if [ "$#" -ne 1 ]; then
    echo "Please specify only the version"
    exit 1
fi

version=$1
docker build -t netice9/apparatchik:$version .
docker push netice9/apparatchik:$version
git tag v$version
git push origin --tags
