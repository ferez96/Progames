venv:
	python3 -m pip install virtualenv
	python3 -m virtualenv venv

venv-dev:
	python3 -m pip install virtualenv
	python3 -m virtualenv venv-dev
	. ./venv-dev/bin/activate; pip install --editable .
	

build: venv
	./scripts/build.sh


dev-init-db: venv-dev
	. ./venv-dev/bin/activate; flask --app progames.server init-db


run-dev-server: venv-dev
	. ./venv-dev/bin/activate; FLASK_RUN_PORT=3000 flask --app progames.server --debug run


.PHONY: build