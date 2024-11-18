#!/bin/sh
openssl ecparam -name prime256v1 -genkey -noout -out ca.key
openssl req -new -x509 -key ca.key -out ca.crt -subj "/O=stiil-test" -days 10
openssl ecparam -name prime256v1 -genkey -noout -out client.key
openssl req -new -key client.key -out client.csr -subj "/CN=user"
openssl x509 -req -days 10 -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt
openssl ecparam -name prime256v1 -genkey -noout -out server.key
openssl req -new -addext "basicConstraints = critical, CA:true" -key server.key -out server.csr -subj "/CN=localhost"
openssl x509 -req -days 10 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt