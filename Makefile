makeFileDir := $(abspath $(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

init:
	npm i -g aws-cdk

build-infra:
	cd $(makeFileDir)/infra && go build

build-queue-consumer:
	cd $(makeFileDir)/queue-consumer && GOOS=linux GOARCH=arm64 go build -o build/bootstrap -tags lambda.norpc

build-queue-producer:
	cd $(makeFileDir)/queue-producer && GOOS=linux GOARCH=arm64 go build -o build/bootstrap -tags lambda.norpc

build-stream-consumer:
	cd $(makeFileDir)/stream-consumer && GOOS=linux GOARCH=arm64 go build -o build/bootstrap -tags lambda.norpc

build-stream-producer:
	cd $(makeFileDir)/stream-producer && GOOS=linux GOARCH=arm64 go build -o build/bootstrap -tags lambda.norpc

build-analyze-test-run:
	cd $(makeFileDir)/analyze-test-run && GOOS=linux GOARCH=arm64 go build -o build/bootstrap -tags lambda.norpc

cdk-ls: build-infra
	cd $(makeFileDir)/infra && cdk ls

cdk-synth: build-infra
	cd $(makeFileDir)/infra && cdk synth

cdk-deploy: build-infra build-queue-consumer build-queue-producer build-stream-consumer build-stream-producer build-analyze-test-run
	cd $(makeFileDir)/infra && cdk deploy

cdk-destroy: build-infra
	cd $(makeFileDir)/infra && cdk destroy
