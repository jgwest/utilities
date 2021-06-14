#!/bin/bash

#CONTAINER_ID=`docker run -d --rm $1 --entrypoint=sleep 10d`
CONTAINER_ID=`docker run -d --entrypoint="sleep" --rm $1 10d`

echo "Container ID is $CONTAINER_ID"

docker exec -u root -it $CONTAINER_ID tar czf /output.tar.gz / 

docker cp $CONTAINER_ID:/output.tar.gz .

docker rm -f $CONTAINER_ID
