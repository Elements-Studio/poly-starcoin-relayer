#!/bin/bash

echo "=== container stopping ==="
docker stop poly-starcoin-relayer
docker rm  poly-starcoin-relayer
echo "=== container stopped ==="