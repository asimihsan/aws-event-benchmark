makeFileDir := $(abspath $(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

build-infra:
	cd $(makeFileDir)/infra && go build

build-queue-consumer:
	cd $(makeFileDir)/queue-consumer && GOOS=linux GOARCH=amd64 go build -o build/queue-consumer

cdk-ls: build-infra
	cd $(makeFileDir)/infra && cdk ls

cdk-synth: build-infra
	cd $(makeFileDir)/infra && cdk synth

cdk-deploy: build-infra build-queue-consumer
	cd $(makeFileDir)/infra && \
		aws-vault exec kittencat-admin --region us-west-2 -- \
			cdk deploy
