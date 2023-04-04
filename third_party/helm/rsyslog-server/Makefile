.PHONY: image push test

push: image
	docker push kodgruvan/rsyslog-server:latest

image:
	docker build . -t kodgruvan/rsyslog-server:latest

