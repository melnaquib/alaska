# Alaska

## Alska Exposes Walrus Storage as an indusry standard s3 API compliant storage. target is to reuse existing codebases and tools to be able to work with Walrus.

## Example use cases;
- ownCloud users - dropbox open source alternative can connect it to s3; Walrus thru Alaska, with no changes at all
- set terraform deployment files to deply to walrus instead of public cloud like a****n or g**le for an easy viabliity evaluation 
- use standard dev tools to access walrus, without any code changes; aws cli tools, dofferent programming languages s3 SDKs etc

## Run demo
docker build -f docker/Dockerfile.go_build .
docker compose up
docker compose ip awscli

#run local
# cp --parents master.toml ~/.seaweed/master.toml
# weed -v=1 master -ip=walrus3 s3 -s3.port=4566 -volumeSizeLimitMB=10


cd weed
make

weed master &
weed filer &
weed s3 &

echo "lock;volume.tier.upload -dest walrus -fullPercent=95 -quietFor=1h;unlock" | weed shell

aws --no-sign-request --endpoint-url="http://localhost:4566" s3 ls nxc01

docker compose up nextcloud

open http://localhost:30080/


##Work in progress
# currently s3 api supports secp256k1 or secp256r1 keys
for future impl we can relay signed message from S3 API headers to proxy / sponsored move contract to use same key for s3 call and walrus storage op.

## future work
- eaasier demo steps
- wrap rust sdk with golnang sdk to expose features like delete, list, partial read/writes, etc.
- support different mode in intermediate storage, and make easy config tools to configure lifecycle, go_thru modes etc. 