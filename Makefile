all: api auth

.PHONY: api auth package deploy

api:
	GOOS=linux go build -o api src/api/main.go

auth:
	GOOS=linux go build -o auth src/auth/main.go

package: api auth
	sam package --template-file template.yaml --s3-bucket $(S3_BUCKET) --output-template-file packaged.yml

deploy:
	sam deploy --template-file packaged.yml --stack-name $(STACK_NAME) --capabilities CAPABILITY_IAM

install: package deploy
