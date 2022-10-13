venv:
	python3 -m pip install virtualenv
	python3 -m virtualenv venv
	. ./venv/bin/activate; pip install -r requirements.txt 

venv-dev:
	python3 -m pip install virtualenv
	python3 -m virtualenv venv-dev
	. ./venv-dev/bin/activate; pip install --editable .
	

build:
	# do nothing

test: venv
	. ./venv/bin/activate; pip install .
	. ./venv/bin/activate; pytest --no-header -vv --durations=10



dev-init-db: venv-dev
	. ./venv-dev/bin/activate; flask --app progames.server init-db


dev-run-server: venv-dev
	. ./venv-dev/bin/activate; FLASK_RUN_PORT=3000 flask --app progames.server --debug run


.PHONY: build test