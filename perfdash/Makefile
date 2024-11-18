all: push

# See deployment.yaml for the version currently running-- bump this ahead before rebuilding!
TAG?=2.48_test5

REPO?=gcr.io/scp-engsrvperfdev-gcp

test: perfdash.go parser.go config.go metrics-downloader-helper.go metrics-downloader.go gcs_metrics_bucket.go s3_metrics_bucket.go
	go test

perfdash: test
	go build -a -installsuffix cgo -ldflags '-w' -o perfdash

run: perfdash
	./perfdash \
		--www \
		--address=0.0.0.0:8080 \
		--builds=20 \
		--configPath=./parser-config.yaml
		--logsBucket=gardener-performance
		--credentialPath=/Users/I524895/myworks/service-account-gcp.json 

# Use buildkit to have "COPY --chmod=" support (availability of it in "regular" docker build depends on docker version).
container:
	DOCKER_BUILDKIT=1 docker build --pull -t $(REPO)/perfdash:$(TAG) .

push: 
	DOCKER_BUILDKIT=1 docker buildx build --platform linux/amd64,linux/arm64 -t $(REPO)/perfdash:$(TAG) . --push

clean:
	rm -f perfdash
