#!/usr/bin/env sh

BUILD_DIR=./build


activate () {
  . ./venv/bin/activate
}

if [ ! -d $BUILD_DIR ]; then
    mkdir $BUILD_DIR
fi

activate
python --version
pip install build
python -m build