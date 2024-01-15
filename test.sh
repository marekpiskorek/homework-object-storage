#!/bin/bash
curl -X POST http://localhost:3000/object/alphanumeric1 -H "Content-Type: application/json" -d '{"objectId": "1"}' --output -
curl -X POST http://localhost:3000/object/2134asdvag322 -H "Content-Type: application/json" -d '{"objectId": "2"}' --output -
curl -X POST http://localhost:3000/object/33333333333333 -H "Content-Type: application/json" -d '{"objectId": "3"}' --output -
curl -X POST http://localhost:3000/object/xxx -H "Content-Type: application/json" -d '{"objectId": "4"}' --output -
curl -X POST http://localhost:3000/object/uyuyuyuy -H "Content-Type: application/json" -d '{"objectId": "5"}' --output -

curl -X GET http://localhost:3000/object/uyuyuyuy --output - # 5
curl -X GET http://localhost:3000/object/xxx --output - # 4
curl -X GET http://localhost:3000/object/33333333333333 --output - # 3
curl -X GET http://localhost:3000/object/2134asdvag322 --output - # 2
curl -X GET http://localhost:3000/object/alphanumeric1 --output - # 1
