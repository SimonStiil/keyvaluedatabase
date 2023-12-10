kubectl -n kvdb-ooc get secret server-tls -o json | jq -r ".data.\"tls.crt\"" |base64 -d >server.crt
kubectl -n kvdb-ooc get secret server-tls -o json | jq -r ".data.\"tls.key\"" |base64 -d >server.key
kubectl -n kvdb-ooc get secret mtls -o json | jq -r ".data.\"tls.crt\"" |base64 -d >client.crt
kubectl -n kvdb-ooc get secret mtls -o json | jq -r ".data.\"tls.key\"" |base64 -d >client.key
kubectl -n kvdb-ooc get secret intermediate-ca -o json | jq -r ".data.\"ca.crt\"" |base64 -d >ca.crt
kubectl -n kvdb-ooc get secret intermediate-ca -o json | jq -r ".data.\"tls.crt\"" |base64 -d >>ca.crt