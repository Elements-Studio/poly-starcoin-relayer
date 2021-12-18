## docker poly-starcoin-relayer

1. stop container，delete old container

`./stop.sh`

2. build docker image

`./build.sh`

3. start container

`./run.sh`

4. check log

`docker logs -f poly-starcoin-relayer`

5. One-click for all above

`./rebuild.sh`

6. inspect a running container.
`docker exec -it <CONTAINER_ID> /bin/bash`
