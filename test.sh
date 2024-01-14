#!/bin/bash
curl -X POST http://localhost:3000/object/1 -H "Content-Type: application/json" -d '{"objectId": "1"}' --output -
curl -X POST http://localhost:3000/object/2 -H "Content-Type: application/json" -d '{"objectId": "2"}' --output -
curl -X POST http://localhost:3000/object/3 -H "Content-Type: application/json" -d '{"objectId": "3"}' --output -
curl -X POST http://localhost:3000/object/4 -H "Content-Type: application/json" -d '{"objectId": "4"}' --output -
curl -X POST http://localhost:3000/object/5 -H "Content-Type: application/json" -d '{"objectId": "5"}' --output -

curl -X GET http://localhost:3000/object/1 --output -
curl -X GET http://localhost:3000/object/2 --output -
curl -X GET http://localhost:3000/object/3 --output -
curl -X GET http://localhost:3000/object/4 --output -
curl -X GET http://localhost:3000/object/5 --output -
