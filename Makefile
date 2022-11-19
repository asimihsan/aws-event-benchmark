makeFileDir := $(abspath $(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

build-infra:
	cd $(makeFileDir)/infra && go build

build-queue-consumer:
	cd $(makeFileDir)/queue-consumer && GOOS=linux GOARCH=arm64 go build -o build/bootstrap -tags lambda.norpc

build-queue-producer:
	cd $(makeFileDir)/queue-producer && GOOS=linux GOARCH=arm64 go build -o build/bootstrap -tags lambda.norpc

build-analyze-test-run:
	cd $(makeFileDir)/analyze-test-run && GOOS=linux GOARCH=arm64 go build -o build/bootstrap -tags lambda.norpc

cdk-ls: build-infra
	cd $(makeFileDir)/infra && cdk ls

cdk-synth: build-infra
	cd $(makeFileDir)/infra && cdk synth

cdk-deploy: build-infra build-queue-consumer build-queue-producer build-analyze-test-run
	cd $(makeFileDir)/infra && \
		aws-vault exec kittencat-admin --region us-west-2 -- \
			cdk deploy
